import { Telegraf } from 'telegraf'
import {
  AccessTokenKeyName,
  TelegramUserID,
  FIBAPIBaseURL,
  LastNoticeTimestampKeyName,
  NoAvailableNoticesErrorMessage,
  NoticeUnavailableErrorMessage,
  RefreshTokenKeyName,
} from './constants'
import { buildNoticeMessage, getHash } from './helpers'
import { UserInfo, Notices } from './models'

function Bot(token) {
  let bot = new Telegraf(token)

  // middlewares
  bot.use(async (ctx, next) => {
    // user authentication
    if (ctx.from.id !== TelegramUserID) {
      return
    }
    await next()
  })

  // calls FIB API with the received authorization code to get tokens, and sends a message to user if succeeded
  bot.authorize = async (authorizationCode) => {
    const accessToken = await authorize(authorizationCode)
    const userInfo = new UserInfo(await (await fetch(`${FIBAPIBaseURL}/jo.json`, {
      headers: { 'Authorization': `Bearer ${accessToken}` },
    })).json())

    await bot.telegram.sendMessage(TelegramUserID, `Hello, ${userInfo.firstName}!`)
  }

  // gets all notices from FIB API
  bot.getNotices = async () => {
    const accessToken = await getAccessToken()
    return new Notices(await (await fetch(`${FIBAPIBaseURL}/jo/avisos.json`, {
      headers: { 'Authorization': `Bearer ${accessToken}` },
    })).json())
  }

  // gets new notices from FIB API
  bot.getNewNotices = async () => {
    const notices = await bot.getNotices()
    if (notices.length === 0) {
      return []
    }

    notices.sort((i, j) => i.publishedAt - j.publishedAt)

    let newNotices = []
    const lastNoticeTimestamp = parseInt(await KV.get(LastNoticeTimestampKeyName))
    for (const notice of notices) {
      if (notice.publishedAt > lastNoticeTimestamp) {
        newNotices.push(notice)
      }
    }

    // updates last notice timestamp in KV
    await KV.put(LastNoticeTimestampKeyName, notices[notices.length - 1].publishedAt.toString())
    return newNotices
  }

  // gets and pushes new notices
  bot.pushNewNotices = async () => {
    const notices = await bot.getNewNotices()
    if (notices.length === 0) {
      return
    }

    for (const notice of notices) {
      const msg = await buildNoticeMessage(notice)
      await bot.telegram.sendMessage(TelegramUserID, msg, { parse_mode: 'HTML' })
    }
  }

  // starts PM session
  bot.start(async (ctx) => await ctx.reply('OK'))

  // generates login (FIB API OAuth authorization) link and sends it to user
  bot.command('login', async (ctx) => {
    const state = await getHash(TelegramUserID.toString())
    const oauthURL = `${FIBAPIBaseURL}/o/authorize/?client_id=${FIBAPI_OAUTH_CLIENT_ID}&redirect_uri=${FIBAPI_REDIRECT_URI}&response_type=code&scope=read&state=${state}`
    await ctx.replyWithHTML(`<a href="${oauthURL}">Authorize Racó Bot</a>`)
  })

  // pulls user info from FIB API and sends it to user
  bot.command('whoami', async (ctx) => {
    const accessToken = await getAccessToken()
    const userInfo = new UserInfo(await (await fetch(`${FIBAPIBaseURL}/jo.json`, {
      headers: { 'Authorization': `Bearer ${accessToken}` },
    })).json())

    await ctx.reply(`${userInfo.firstName} ${userInfo.lastNames}`)
  })

  // (for notices debugging only)
  bot.command('debug', async (ctx) => {
    const notices = await bot.getNotices()
    if (notices.length === 0) {
      await ctx.replyWithHTML(NoAvailableNoticesErrorMessage)
      return
    }

    const debugNoticeId = parseInt(ctx.message.text.split(' ')[1])
    for (const notice of notices) {
      if (notice.id === debugNoticeId) {
        const msg = await buildNoticeMessage(notice)
        await ctx.replyWithHTML(msg)
        return
      }
    }

    await ctx.replyWithHTML(NoticeUnavailableErrorMessage)
  })

  // (for notices debugging only)
  bot.command('test', async (ctx) => {
    const notices = await bot.getNotices()
    if (notices.length === 0) {
      await ctx.replyWithHTML(NoAvailableNoticesErrorMessage)
      return
    }

    notices.sort((i, j) => i.publishedAt - j.publishedAt)
    const msg = await buildNoticeMessage(notices[notices.length - 1])
    await ctx.replyWithHTML(msg)
  })

  return bot
}

// authorize requests access token from FIB API with authorization code
async function authorize(authorizationCode) {
  const data = await (await fetch(`${FIBAPIBaseURL}/o/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      'grant_type': 'authorization_code',
      'code': authorizationCode,
      'redirect_uri': FIBAPI_REDIRECT_URI,
      'client_id': FIBAPI_OAUTH_CLIENT_ID,
      'client_secret': FIBAPI_OAUTH_CLIENT_SECRET,
    }),
  })).json()
  if (!data || !data.access_token || data.access_token.length !== 30
    || !data.refresh_token || data.refresh_token.length !== 30) {
    throw new Error(`[FIB API] Invalid OAuth authorizing response: ${JSON.stringify(data)}`)
  }

  await KV.put(AccessTokenKeyName, data.access_token, { expirationTtl: 36000 - 30 })
  await KV.put(RefreshTokenKeyName, data.refresh_token)

  return data.access_token
}

// refreshAuthorization refreshes the access token and refresh token from FIB API with refresh token
async function refreshAuthorization(refreshToken) {
  const data = await (await fetch(`${FIBAPIBaseURL}/o/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      'grant_type': 'refresh_token',
      'refresh_token': refreshToken,
      'client_id': FIBAPI_OAUTH_CLIENT_ID,
      'client_secret': FIBAPI_OAUTH_CLIENT_SECRET,
    }),
  })).json()
  if (!data || !data.access_token || data.access_token.length !== 30
    || !data.refresh_token || data.refresh_token.length !== 30) {
    throw new Error('[FIB API] Invalid OAuth refreshing response')
  }

  await KV.put(AccessTokenKeyName, data.access_token, { expirationTtl: 36000 - 30 })
  await KV.put(RefreshTokenKeyName, data.refresh_token)

  return data.access_token
}

// gets FIB API access token from KV, and requests a new one when expired
async function getAccessToken() {
  let accessToken = await KV.get(AccessTokenKeyName)
  if (!accessToken || accessToken.length !== 30) {
    const refreshToken = await KV.get(RefreshTokenKeyName)
    if (!refreshToken || refreshToken.length !== 30) {
      throw new Error('Invalid FIB API tokens in KV')
    }
    accessToken = await refreshAuthorization(refreshToken)
  }
  return accessToken
}

export { Bot }

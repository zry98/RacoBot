import { Telegraf } from 'telegraf'
import {
  BotUserID,
  FIBAPIBaseURL,
  AccessTokenKeyName,
  RefreshTokenKeyName,
  LastNoticeTimestampKeyName,
} from './constants'
import { buildNoticeMessage, getHash } from './helpers'

function Bot(token) {
  let bot = new Telegraf(token)

  // middlewares
  bot.use(async (ctx, next) => {
    // user authentication
    if (ctx.from.id !== BotUserID) {
      return
    }
    await next()
  })

  // calls FIB API with the received authorization code to get tokens, and sends a message to user if succeeded
  bot.authorize = async (authorizationCode) => {
    const accessToken = await authorize(authorizationCode)
    const data = await (await fetch(`${FIBAPIBaseURL}/jo.json`, {
      headers: { 'Authorization': `Bearer ${accessToken}` },
    })).json()
    if (!data || !data.nom) {
      throw new Error('[FIB API] Invalid userinfo response')
    }

    await bot.telegram.sendMessage(BotUserID, `Hello, ${data.nom}!`)
  }

  // pulls notices from FIB API and forwards those newer ones to user
  bot.pushNewNotices = async () => {
    const accessToken = await getAccessToken()
    const data = await (await fetch(`${FIBAPIBaseURL}/jo/avisos.json`, {
      headers: { 'Authorization': `Bearer ${accessToken}` },
    })).json()
    if (!data || !data.results) {
      throw new Error('[FIB API] Invalid notices response')
    }

    const lastNoticeTimestamp = parseInt(await KV.get(LastNoticeTimestampKeyName))
    let newLastNoticeTimestamp = lastNoticeTimestamp
    for (const notice of data.results) {
      const noticeTimestamp = Date.parse(notice.data_modificacio)
      if (noticeTimestamp > lastNoticeTimestamp) {
        if (noticeTimestamp > newLastNoticeTimestamp) {
          newLastNoticeTimestamp = noticeTimestamp
        }
        const msg = await buildNoticeMessage(notice)
        await bot.telegram.sendMessage(BotUserID, msg, { parse_mode: 'HTML' })
      }
    }
    await KV.put(LastNoticeTimestampKeyName, newLastNoticeTimestamp.toString())
  }

  // starts PM session
  bot.start(async (ctx) => await ctx.reply('OK'))

  // generates login (FIB API OAuth authorization) link and sends it to user
  bot.command('login', async (ctx) => {
    const state = await getHash(BotUserID.toString())
    const oauthURL = `${FIBAPIBaseURL}/o/authorize/?client_id=${FIBAPI_OAUTH_CLIENT_ID}&redirect_uri=${FIBAPI_REDIRECT_URI}&response_type=code&scope=read&state=${state}`
    await ctx.replyWithHTML(`<a href="${oauthURL}">Authorize Racó Bot</a>`)
  })

  // pulls user info from FIB API and sends it to user
  bot.command('whoami', async (ctx) => {
    const accessToken = await getAccessToken()
    const data = await (await fetch(`${FIBAPIBaseURL}/jo.json`, {
      headers: { 'Authorization': `Bearer ${accessToken}` },
    })).json()
    if (!data || !data.nom || !data.cognoms || !data.email) {
      throw new Error('[FIB API] Invalid userinfo response')
    }

    await ctx.reply(`${data.nom} ${data.cognoms}`)
  })

  // (for notices debugging only)
  bot.command('debug', async (ctx) => {
    const accessToken = await getAccessToken()
    const data = await (await fetch(`${FIBAPIBaseURL}/jo/avisos.json`, {
      headers: { 'Authorization': `Bearer ${accessToken}` },
    })).json()
    if (!data || !data.results) {
      throw new Error('[FIB API] Invalid notices response')
    }

    const debugNoticeId = parseInt(ctx.message.text.split(' ')[1])
    for (const notice of data.results) {
      if (notice.id === debugNoticeId) {
        const msg = await buildNoticeMessage(notice)
        await ctx.replyWithHTML(msg)
        break
      }
    }
  })

  bot.command('test', async (ctx) => {
    await bot.pushNewNotices()
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

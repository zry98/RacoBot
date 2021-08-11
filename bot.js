import { Telegraf } from 'telegraf'
import { FIBAPI_BASE_URL, LAST_NOTICE_TIMESTAMP_KEY_NAME } from './constants'
import { buildNoticeMessage, authorize, getAccessToken } from './helpers'

function Bot(token) {
  let bot = new Telegraf(token)

  // calls FIB API with the received authorization code to get tokens, and sends a message to user if succeeded
  bot.authorize = async (authorizationCode) => {
    const accessToken = await authorize(authorizationCode)
    const data = await (await fetch(`${FIBAPI_BASE_URL}/jo.json`, {
      headers: { 'Authorization': `Bearer ${accessToken}` },
    })).json()
    if (!data || !data.nom) {
      throw new Error('[FIB API] Invalid userinfo response')
    }

    await bot.telegram.sendMessage(USER_ID, `Hello, ${data.nom}!`)
  }

  // pulls notices from FIB API and forwards those newer ones to user
  bot.pushNewNotices = async () => {
    const accessToken = await getAccessToken()
    const data = await (await fetch(`${FIBAPI_BASE_URL}/jo/avisos.json`, {
      headers: { 'Authorization': `Bearer ${accessToken}` },
    })).json()
    if (!data || !data.results) {
      throw new Error('[FIB API] Invalid notices response')
    }

    const lastNoticeTimestamp = parseInt(await KV.get(LAST_NOTICE_TIMESTAMP_KEY_NAME))
    let newLastNoticeTimestamp = lastNoticeTimestamp
    for (const notice of data.results) {
      const noticeTimestamp = Date.parse(notice.data_modificacio)
      if (noticeTimestamp > lastNoticeTimestamp) {
        if (noticeTimestamp > newLastNoticeTimestamp) {
          newLastNoticeTimestamp = noticeTimestamp
        }
        const msg = await buildNoticeMessage(notice)
        await bot.telegram.sendMessage(USER_ID, msg, { parse_mode: 'HTML' })
      }
    }
    await KV.put(LAST_NOTICE_TIMESTAMP_KEY_NAME, newLastNoticeTimestamp.toString())
  }

  // starts PM session
  bot.start(async (ctx) => await ctx.reply('OK'))

  // generates login (FIB API OAuth authorization) link and sends it to user
  bot.command('login', async (ctx) => {
    const oauthState = Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15)
    const oauthURL = `${FIBAPI_BASE_URL}/o/authorize/?client_id=${FIBAPI_OAUTH_CLIENT_ID}&redirect_uri=${FIBAPI_REDIRECT_URI}&response_type=code&scope=read&state=${oauthState}`
    await ctx.replyWithHTML(`<a href="${oauthURL}">Authorize Racó Bot</a>`)
  })

  // pulls user info from FIB API and sends it to user
  bot.command('whoami', async (ctx) => {
    const accessToken = await getAccessToken()
    const data = await (await fetch(`${FIBAPI_BASE_URL}/jo.json`, {
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
    const data = await (await fetch(`${FIBAPI_BASE_URL}/jo/avisos.json`, {
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

export { Bot }

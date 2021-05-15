import {Telegraf} from 'telegraf'
import {FIBAPI_BASE_URL} from './constants'
import {buildNoticeMessage, authorize, getAccessToken} from './helpers'

function Bot(token) {
  let bot = new Telegraf(token)

  // calls FIB API with received authorization code to get access token, and sends message to user if succeeded
  bot.authorize = async (authorizationCode) => {
    const accessToken = await authorize(authorizationCode)
    const data = await (await fetch(`${FIBAPI_BASE_URL}/jo/`, {
      headers: {
        'Accept': 'application/json',
        'Authorization': `Bearer ${accessToken}`,
      },
    })).json()
    if (!data || !data.nom) {
      throw new Error('[FIB API] Invalid userinfo response')
    }

    await bot.telegram.sendMessage(USER_ID, `Hello, ${data.nom}!`)
  }

  // pulls notices from FIB API and forwards those newer ones to user
  bot.pushNewNotices = async () => {
    const accessToken = await getAccessToken()
    const data = await (await fetch(`${FIBAPI_BASE_URL}/jo/avisos/`, {
      headers: {
        'Accept': 'application/json',
        'Authorization': `Bearer ${accessToken}`,
      },
    })).json()
    if (!data || !data.results) {
      throw new Error('[FIB API] Invalid notices response')
    }

    const lastNoticeId = parseInt(await KV.get('last_notice_id'))
    let newLastNoticeId = lastNoticeId
    for (const notice of data.results) {
      if (notice.id > lastNoticeId) {
        if (notice.id > newLastNoticeId) {
          newLastNoticeId = notice.id
        }
        const msg = await buildNoticeMessage(notice.codi_assig, notice.titol, notice.text)
        await bot.telegram.sendMessage(USER_ID, msg, {parse_mode: 'HTML'})
      }
    }
    await KV.put('last_notice_id', newLastNoticeId.toString())
  }

  // starts PM session
  bot.start((ctx) => ctx.reply('OK'))

  // generates login (FIB API OAuth authorization) link and sends it to user
  bot.command('login', (ctx) => {
    const oauthState = Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15)
    const oauthURL = `${FIBAPI_BASE_URL}/o/authorize/?client_id=${FIBAPI_OAUTH_CLIENT_ID}&redirect_uri=${FIBAPI_REDIRECT_URI}&response_type=code&scope=read&state=${oauthState}`
    ctx.replyWithHTML(`<a href="${oauthURL}">Authorize RacóBot</a>`)
  })

  // pulls user info from FIB API and sends it to user
  bot.command('whoami', async (ctx) => {
    const accessToken = await getAccessToken()
    const data = await (await fetch(`${FIBAPI_BASE_URL}/jo/`, {
      headers: {
        'Accept': 'application/json',
        'Authorization': `Bearer ${accessToken}`,
      },
    })).json()
    if (!data || !data.nom || !data.cognoms || !data.email) {
      throw new Error('[FIB API] Invalid userinfo response')
    }

    await ctx.reply(`${data.nom} ${data.cognoms}\n\n${data.email}`)
  })

  // (for notices debugging only)
  bot.command('debug', async (ctx) => {
    const accessToken = await getAccessToken()
    const data = await (await fetch(`${FIBAPI_BASE_URL}/jo/avisos/`, {
      headers: {
        'Accept': 'application/json',
        'Authorization': `Bearer ${accessToken}`,
      },
    })).json()
    if (!data || !data.results) {
      throw new Error('[FIB API] Invalid notices response')
    }

    const debugNoticeId = parseInt(ctx.message.text.split(' ')[1])
    for (const notice of data.results) {
      if (notice.id === debugNoticeId) {
        const msg = await buildNoticeMessage(notice.codi_assig, notice.titol, notice.text)
        await ctx.replyWithHTML(msg)
      }
    }
  })

  return bot
}

export {Bot}

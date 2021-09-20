import { Bot } from './bot'
import {
  TelegramBotAPIToken,
  AuthorizedHTML,
  TelegramBotWebhookPath,
  TelegramUserID,
  FIBAPIOAuthRedirectURLPath,
} from './constants'
import { getHash, timingSafeEqual } from './helpers'

const bot = Bot(TelegramBotAPIToken)

addEventListener('fetch', event => {
  event.respondWith(handleRequest(event.request).then(resp => {
    return resp
  }).catch(e => {
    console.log(e)
    return new Response('ERROR')
  }))
})

async function handleRequest(request) {
  const url = new URL(request.url)
  if (request.method === 'POST' && url.pathname === TelegramBotWebhookPath) {  // Telegram update
    await bot.handleUpdate(await request.json())
    return new Response('OK')

  } else if (request.method === 'GET' && url.pathname.startsWith(FIBAPIOAuthRedirectURLPath)) {  // FIB API OAuth callback
    const code = url.searchParams.get('code'), state = url.searchParams.get('state')
    if (!code || !state || code.length !== 30 ||
      !timingSafeEqual(state, await getHash(TelegramUserID.toString()))) {
      throw new Error('Invalid OAuth callback request')
    }

    await bot.authorize(code)
    return new Response(AuthorizedHTML, {
      headers: { 'Content-Type': 'text/html;charset=UTF-8' },
    })
  }

  throw new Error('Invalid request')
}

addEventListener('scheduled', event => {
  event.waitUntil(handleScheduled(event))
})

// handles scheduled jobs of checking for new notices and pushing them to user
async function handleScheduled(event) {
  await bot.pushNewNotices()
}

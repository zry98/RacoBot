import {Bot} from './bot'
import {AUTHORIZED_HTML} from './constants'

const bot = Bot(TELEGRAM_BOT_API_TOKEN)

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
  if (request.method === 'POST' && url.pathname === '/bot') {  // Telegram update
    await bot.handleUpdate(await request.json())
    return new Response('OK')

  } else if (request.method === 'GET' && url.pathname.startsWith('/o/authorize')) {  // FIB API OAuth callback
    const authorizationCode = url.searchParams.get('code')
    if (!authorizationCode || authorizationCode.length !== 30) {
      throw new Error('[FIB API] Invalid OAuth callback request')
    }

    await bot.authorize(authorizationCode)

    return new Response(AUTHORIZED_HTML, {
      headers: {
        'Content-Type': 'text/html;charset=UTF-8',
      },
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

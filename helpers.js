import { decode } from 'html-entities'  // for un-escaping HTML entities like `&#39;` since HTMLRewriter can't do that
import { FIBAPI_BASE_URL, FIBAPI_ACCESS_TOKEN_KEY_NAME, FIBAPI_REFRESH_TOKEN_KEY_NAME } from './constants'

// buildNoticeMessage formats a notice to a proper string ready to be sent by bot
async function buildNoticeMessage(notice) {
  const supportedTagNames = ['a', 'b', 'strong', 'i', 'em', 'u', 'ins', 's', 'strike', 'del', 'code', 'pre']
  const messageMaxLength = 4096

  class ElementContentHandler {
    element(e) {
      if (e.tagName === 'br') {
        // Telegram doesn't support <br> but \n
        e.replace('\n')
      } else if (e.tagName === 'li') {
        // Telegram doesn't support <ul> & <li>, so add a `- ` at the beginning as an indicator
        e.before('- ')
        e.after('\n')  // newline after each entry
      } else if (e.tagName === 'div' && e.getAttribute('class') === 'extraInfo') {
        // add newlines before exam info titles
        e.before('\n')
      } else if (e.tagName === 'span' && e.getAttribute('id') === 'horaExamen') {
        // add newlines after exam time data
        e.after('\n')
      } else if (e.tagName === 'span' && e.getAttribute('class') === 'label') {
        // italicize info titles
        e.tagName = 'i'
        e.removeAttribute('class')
        e.prepend('- ')
      } else if (e.tagName === 'span' && e.getAttribute('style') === 'text-decoration:underline') {
        // underlines
        e.tagName = 'u'
        e.removeAttribute('style')
      }
      if (!supportedTagNames.includes(e.tagName)) {
        // strip all the other tags since Telegram doesn't support them
        e.removeAndKeepContent()
      }
    }
  }

  let result = ''
  if (notice.text !== '') {
    result = '\n\n' + decode(await ((new HTMLRewriter().on('*', new ElementContentHandler()).transform(new Response(notice.text))).text()))
  }
  result = `[${notice.codi_assig}] <b>${notice.titol}</b>${result}`

  if (result.length > messageMaxLength) {
    result = `[${notice.codi_assig}] <b>${notice.titol}</b>\n\nSorry, but this message is too long to be sent through Telegram, please view it through <a href="https://raco.fib.upc.edu/avisos/veure.jsp?assig=GRAU-${notice.codi_assig}&id=${notice.id}">this link</a>.`
  }

  return result
}

// authorize requests access token from FIB API with authorization code
async function authorize(authorizationCode) {
  const data = await (await fetch(`${FIBAPI_BASE_URL}/o/token`, {
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
    throw new Error('[FIB API] Invalid OAuth authorizing response')
  }

  await KV.put(FIBAPI_ACCESS_TOKEN_KEY_NAME, data.access_token, { expirationTtl: 36000 - 30 })
  await KV.put(FIBAPI_REFRESH_TOKEN_KEY_NAME, data.refresh_token)

  return data.access_token
}

// refreshAuthorization refreshes the access token and refresh token from FIB API with refresh token
async function refreshAuthorization(refreshToken) {
  const data = await (await fetch(`${FIBAPI_BASE_URL}/o/token`, {
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

  await KV.put(FIBAPI_ACCESS_TOKEN_KEY_NAME, data.access_token, { expirationTtl: 36000 - 30 })
  await KV.put(FIBAPI_REFRESH_TOKEN_KEY_NAME, data.refresh_token)

  return data.access_token
}

// gets FIB API access token from KV, and requests a new one when expired
async function getAccessToken() {
  let accessToken = await KV.get(FIBAPI_ACCESS_TOKEN_KEY_NAME)
  if (!accessToken || accessToken.length !== 30) {
    const refreshToken = await KV.get(FIBAPI_REFRESH_TOKEN_KEY_NAME)
    if (!refreshToken || refreshToken.length !== 30) {
      throw new Error('Invalid FIBAPI.ACCESS_TOKEN and FIBAPI.REFRESH_TOKEN in KV')
    }
    accessToken = await refreshAuthorization(refreshToken)
  }
  return accessToken
}

export { buildNoticeMessage, authorize, getAccessToken }

import {decode} from 'html-entities'  // for un-escaping HTML entities like `&#39;` since HTMLRewriter can't do that
import {FIBAPI_BASE_URL} from './constants'

// builds notice message for Telegram from HTML in FIB API response
async function buildNoticeMessage(subjectCode, title, text) {
  const supportedTags = ['a', 'b', 'i', 'u', 's', 'code', 'pre']

  class UserElementHandler {
    element(element) {
      if (element.tagName === 'br') {
        // Telegram doesn't support <br> but \n
        element.replace('\n')
      } else if (element.tagName === 'span' && element.getAttribute('class') === 'label') {
        // italicize info titles
        element.tagName = 'i'
        element.removeAttribute('class')
        element.prepend('- ')
      } else if (element.tagName === 'div' && element.getAttribute('class') === 'extraInfo') {
        // add newlines before exam info titles  // TODO: check if notice 117170 has a typo of newline in aulesExamen
        element.before('\n')
      }
      if (!supportedTags.includes(element.tagName)) {
        // strip all the other tags since Telegram doesn't support them
        element.removeAndKeepContent()
      }
    }
  }

  let msg = new HTMLRewriter().
      // TODO: use multiple handlers instead of matching all tags in one?
      // on('div[class="examen"] > div[class="extraInfo"] > *', new ExtraInfoElementHandler()).
      on('*', new UserElementHandler()).
      transform(new Response(text))
  msg = decode(await (msg.text()))

  return `[${subjectCode}] <b>${title}</b>\n\n${msg}`
}

// requests access token from FIB API with authorization code or refresh token
async function authorize(authorizationCode = null, refreshToken = null) {
  let reqBody
  if (authorizationCode !== null && refreshToken === null) {  // first time authorization using authorization_code
    reqBody = new URLSearchParams({
      'grant_type': 'authorization_code',
      'redirect_uri': FIBAPI_REDIRECT_URI,
      'code': authorizationCode,
      'client_id': FIBAPI_OAUTH_CLIENT_ID,
      'client_secret': FIBAPI_OAUTH_CLIENT_SECRET,
    })
  } else if (authorizationCode === null && refreshToken !== null) {  // refresh authorization using refresh_token
    reqBody = new URLSearchParams({
      'grant_type': 'refresh_token',
      'refresh_token': refreshToken,
      'client_id': FIBAPI_OAUTH_CLIENT_ID,
      'client_secret': FIBAPI_OAUTH_CLIENT_SECRET,
    })
  } else {
    throw new Error('[FIB API] Wrong authorization request parameters')
  }

  const data = await (await fetch(`${FIBAPI_BASE_URL}/o/token`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
    },
    body: reqBody,
  })).json()
  if (!data || !data.access_token || data.access_token.length !== 30
      || !data.refresh_token || data.refresh_token.length !== 30) {
    throw new Error('[FIB API] Invalid OAuth token response')
  }

  await KV.put('fibapi.access_token', data.access_token, {expirationTtl: 36000})
  await KV.put('fibapi.refresh_token', data.refresh_token)

  return data.access_token
}

// gets FIB API access token from KV, and requests a new one when expired
async function getAccessToken() {
  let accessToken = await KV.get('fibapi.access_token')
  while (!accessToken || accessToken.length !== 30) {
    let refreshToken = await KV.get('fibapi.refresh_token')
    if (!refreshToken || refreshToken.length !== 30) {
      throw new Error('Invalid FIBAPI.ACCESS_TOKEN and FIBAPI.REFRESH_TOKEN in KV')
    }
    accessToken = await authorize(null, refreshToken)
  }
  return accessToken
}

export {buildNoticeMessage, authorize, getAccessToken}

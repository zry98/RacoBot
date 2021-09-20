const FIBAPIBaseURL = 'https://api.fib.upc.edu/v2'
const FIBAPILoginRedirectBaseURL = 'https://api.fib.upc.edu/v2/accounts/login/?next='
const RacoBaseURL = 'https://raco.fib.upc.edu'
const AuthorizedHTML = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>Racó Bot</title></head><body><h1>Authorized</h1><p>You can close the browser and return to Telegram.</p>
<script type="text/javascript">
  window.addEventListener('load', _ => {
    window.location.href='tg://resolve?domain=${TELEGRAM_BOT_USERNAME}'
  })
</script></body></html>`

const LastNoticeTimestampKeyName = 'last_notice_timestamp'
const AccessTokenKeyName = 'fibapi.access_token'
const RefreshTokenKeyName = 'fibapi.refresh_token'

const TelegramUserID = parseInt(TELEGRAM_USER_ID)
const TelegramBotAPIToken = TELEGRAM_BOT_API_TOKEN
const TelegramBotWebhookPath = '/' + TELEGRAM_BOT_WEBHOOK_PATH
const FIBAPIOAuthRedirectURLPath = (new URL(FIBAPI_REDIRECT_URI)).pathname

const NoticeUnavailableErrorMessage = '<i>Notice unavailable</i>'
const NoAvailableNoticesErrorMessage = '<i>No available notices</i>'

export {
  FIBAPIBaseURL,
  FIBAPILoginRedirectBaseURL,
  RacoBaseURL,
  AuthorizedHTML,
  LastNoticeTimestampKeyName,
  AccessTokenKeyName,
  RefreshTokenKeyName,
  TelegramUserID,
  TelegramBotAPIToken,
  TelegramBotWebhookPath,
  FIBAPIOAuthRedirectURLPath,
  NoticeUnavailableErrorMessage,
  NoAvailableNoticesErrorMessage
}

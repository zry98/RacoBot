const FIBAPIBaseURL = 'https://api.fib.upc.edu/v2'
const FIBAPILoginRedirectBaseURL = 'https://api.fib.upc.edu/v2/accounts/login/?next='
const AuthorizedHTML = `<!DOCTYPE html>
<body>
  <h1>Authorized</h1>
</body>` // TODO: make it nicer

const LastNoticeTimestampKeyName = 'last_notice_timestamp'
const AccessTokenKeyName = 'fibapi.access_token'
const RefreshTokenKeyName = 'fibapi.refresh_token'

const TelegramBotAPIToken = TELEGRAM_BOT_API_TOKEN
const TelegramBotWebhookPath = '/' + TELEGRAM_BOT_WEBHOOK_PATH
const BotUserID = parseInt(TELEGRAM_BOT_USER_ID)
const FIBAPIOAuthRedirectURLPath = (new URL(FIBAPI_REDIRECT_URI)).pathname

const NoticeUnavailableErrorMessage = '<i>Notice unavailable</i>'
const NoAvailableNoticesErrorMessage = '<i>No available notices</i>'

export {
  FIBAPIBaseURL,
  FIBAPILoginRedirectBaseURL,
  AuthorizedHTML,
  LastNoticeTimestampKeyName,
  AccessTokenKeyName,
  RefreshTokenKeyName,
  TelegramBotAPIToken,
  TelegramBotWebhookPath,
  BotUserID,
  FIBAPIOAuthRedirectURLPath,
  NoticeUnavailableErrorMessage,
  NoAvailableNoticesErrorMessage
}

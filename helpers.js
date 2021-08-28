import { decode } from 'html-entities'  // for un-escaping HTML entities like `&#39;` since HTMLRewriter can't do that

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
    result = await ((new HTMLRewriter().on('*', new ElementContentHandler()).transform(new Response(notice.text))).text())
    result = '\n\n' + decode(result)
  }
  result = `[${notice.subjectCode}] <b>${notice.title}</b>${result}`

  // attachments
  if (notice.attachments.length > 0) {
    let sb = ''
    for (const attachment of notice.attachments) {
      const fileSize = byteCountIEC(attachment.size)
      sb += `<a href="${attachment.redirectURL}">${attachment.name}</a>  (${fileSize})\n`
    }
    const noun = notice.attachments.length > 1 ? 'attachments' : 'attachment'
    result = `${result}\n\n<i>- With ${notice.attachments.length} ${noun}:</i>\n${sb}`
  }

  if (result.length > messageMaxLength) {
    // send Racó notice URL instead if message length exceeds the limit of 4096 characters
    result = `[${notice.subjectCode}] <b>${notice.title}</b>\n\nSorry, but this message is too long to be sent through Telegram, please view it through <a href="https://raco.fib.upc.edu/avisos/veure.jsp?assig=GRAU-${notice.subjectCode}&id=${notice.id}">this link</a>.`
  }

  return result
}

// byteCountIEC returns the human-readable file size of the given bytes count
function byteCountIEC(b) {
  const unit = 1024
  if (b < unit) {
    return `${b} B`
  }
  let div = unit, exp = 0
  for (let n = b / unit; n >= unit; n /= unit) {
    div *= unit
    exp++
  }
  return `${(b / div).toFixed(1)} ${'KMGTPE'[exp]}iB`
}

// getHash returns the base64 encoded SHA-256 hash of the given data
async function getHash(data) {
  const encoder = new TextEncoder()
  // return Array.from(new Uint8Array(
  //   await crypto.subtle.digest('SHA-256', encoder.encode(data)))
  // ).map(b => b.toString(16).padStart(2, '0')).join('')
  return btoa(String.fromCharCode(...new Uint8Array(
    await crypto.subtle.digest('SHA-256', encoder.encode(data))
  )))
}

// timingSafeEqual compares the given two payloads in a timing-safe manner
function timingSafeEqual(a, b) {
  const strA = String(a)
  let strB = String(b)
  const lenA = strA.length
  let result = 0

  if (lenA !== strB.length) {
    strB = strA
    result = 1
  }

  for (let i = 0; i < lenA; i++) {
    result |= (strA.charCodeAt(i) ^ strB.charCodeAt(i))
  }

  return result === 0
}

export { buildNoticeMessage, getHash, timingSafeEqual }

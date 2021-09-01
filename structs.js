import { FIBAPILoginRedirectBaseURL } from './constants'

// UserInfo represents a user's information API response
// Endpoint: /jo
function UserInfo(o) {
  for (const field of ['nom', 'cognoms', 'email']) {
    if (!(field in o)) {
      throw new Error('[FIB API] Invalid UserInfo response')
    }
  }
  return {
    username: o.username,
    firstName: o.nom,
    lastNames: o.cognoms
  }
}

// Notice represents a single notice in a NoticesResponse API response
function Notice(o) {
  for (const field of ['id', 'titol', 'codi_assig', 'text', 'data_insercio', 'data_modificacio', 'data_caducitat', 'adjunts']) {
    if (!(field in o)) {
      throw new Error('[FIB API] Invalid Notice')
    }
  }
  return {
    id: o.id,
    title: o.titol,
    subjectCode: o.codi_assig,
    text: o.text,
    createdAt: Date.parse(o.data_insercio),
    modifiedAt: Date.parse(o.data_modificacio),
    expiresAt: Date.parse(o.data_caducitat),
    attachments: o.adjunts.map(attachment => new Attachment(attachment)),
  }
}

// Attachment represents a single attachment in a Notice's attachments
function Attachment(o) {
  for (const field of ['tipus_mime', 'nom', 'url', 'data_modificacio', 'mida']) {
    if (!(field in o)) {
      throw new Error('[FIB API] Invalid Attachment')
    }
  }
  return {
    mimeTypes: o.tipus_mime,
    name: o.nom,
    url: o.url,
    modifiedAt: Date.parse(o.data_modificacio),
    size: o.mida,
    // redirectURL represents the attachment's FIB API login redirect URL
    // it's useful since FIB API cookies on the user's browser will expire, accessing an attachment's original URL after that will get an `Unauthorized` response
    redirectURL: FIBAPILoginRedirectBaseURL + encodeURIComponent(o.url),
  }
}

// Notices represents a user's notices API response
// Endpoint: /jo/avisos
function Notices(o) {
  for (const field of ['count', 'results']) {
    if (!(field in o)) {
      throw new Error('[FIB API] Invalid Notices response')
    }
  }
  return o.results.map(notice => new Notice(notice))
}

export { UserInfo, Notices }

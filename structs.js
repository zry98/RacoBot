import { FIBAPILoginRedirectBaseURL } from './constants'

// UserInfo represents a user's information API response
// Endpoint: /jo
function UserInfo(o) {
  if (!o || !o.nom || !o.cognoms || !o.email) {
    throw new Error('[FIB API] Invalid UserInfo response')
  }
  return {
    username: o.username,
    firstName: o.nom,
    lastNames: o.cognoms
  }
}

// Notice represents a single notice in a NoticesResponse API response
function Notice(o) {
  if (!o.id || !o.titol || !o.codi_assig || !o.text || !o.data_insercio || !o.data_modificacio || !o.data_caducitat || !o.adjunts) {
    throw new Error('[FIB API] Invalid Notices response')
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
  if (!o.tipus_mime || !o.nom || !o.url || !o.data_modificacio || !o.mida) {
    throw new Error('[FIB API] Invalid Notices response')
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
  if (!o || !o.count || !o.results) {
    throw new Error('[FIB API] Invalid Notices response')
  }
  return o.results.map(notice => new Notice(notice))
}

export { UserInfo, Notices }

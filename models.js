import { FIBAPILoginRedirectBaseURL } from './constants'

// UserInfo represents a user's information API response
// Endpoint: /jo.json
class UserInfo {
  constructor(o) {
    for (const field of ['nom', 'cognoms', 'email']) {
      if (!(field in o)) {
        throw new Error('[FIB API] Invalid UserInfo response')
      }
    }

    this.username = o.username
    this.firstName = o.nom
    this.lastNames = o.cognoms
  }
}

// Notices represents a user's notices API response
// Endpoint: /jo/avisos.json
function Notices(o) {
  for (const field of ['count', 'results']) {
    if (!(field in o)) {
      throw new Error('[FIB API] Invalid Notices response')
    }
  }
  return o.results.map(notice => new Notice(notice))
}

// Notice represents a single notice in a NoticesResponse API response
class Notice {
  constructor(o) {
    for (const field of ['id', 'titol', 'codi_assig', 'text', 'data_insercio', 'data_modificacio', 'data_caducitat', 'adjunts']) {
      if (!(field in o)) {
        throw new Error('[FIB API] Invalid Notice')
      }
    }

    this.id = o.id
    this.title = o.titol
    this.subjectCode = o.codi_assig
    this.text = o.text
    this.createdAt = Date.parse(o.data_insercio)
    this.modifiedAt = Date.parse(o.data_modificacio)
    this.publishedAt = this.modifiedAt < this.createdAt ? this.createdAt : this.modifiedAt
    this.expiresAt = Date.parse(o.data_caducitat)
    this.attachments = o.adjunts.map(attachment => new Attachment(attachment))
  }
}

// Attachment represents a single attachment in a Notice's attachments
class Attachment {
  constructor(o) {
    for (const field of ['tipus_mime', 'nom', 'url', 'data_modificacio', 'mida']) {
      if (!(field in o)) {
        throw new Error('[FIB API] Invalid Attachment')
      }
    }

    this.mimeTypes = o.tipus_mime
    this.name = o.nom
    this.url = o.url
    this.modifiedAt = Date.parse(o.data_modificacio)
    this.size = o.mida
    // redirectURL represents the attachment's FIB API login redirect URL
    // it's useful since FIB API cookies on the user's browser will expire, accessing an attachment's original URL after that will get an `Unauthorized` response
    this.redirectURL = FIBAPILoginRedirectBaseURL + encodeURIComponent(o.url)
  }
}

// Schedule represents a user's schedule API response
// Endpoint: /jo/classes.json
function Schedule(o) {
  for (const field of ['count', 'results']) {
    if (!(field in o)) {
      throw new Error('[FIB API] Invalid Schedule response')
    }
  }
  return o.results.map(class_ => new Class(class_))
}

// Class represents a single class in a ScheduleResponse API response
class Class {
  constructor(o) {
    for (const field of ['codi_assig', 'grup', 'dia_setmana', 'inici', 'durada', 'tipus', 'aules']) {
      if (!(field in o)) {
        throw new Error('[FIB API] Invalid Class')
      }
    }

    this.subjectCode = o.codi_assig
    this.group = o.grup
    this.dayOfWeek = parseInt(o.dia_setmana)
    this.startTime = o.inici
    this.duration = parseInt(o.durada)
    this.types = o.tipus
    this.classrooms = o.aules
  }
}

export { UserInfo, Notices, Schedule }

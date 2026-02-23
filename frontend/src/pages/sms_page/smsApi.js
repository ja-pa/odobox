import * as Backend from '../../../wailsjs/go/main/App'
import { toError } from '../../errorUtils'

const GSM_BASIC = "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞ !\"#¤%&'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"
const GSM_EXT = '^{}\\[~]|€'

function gsmCharUnits(message) {
  const basic = new Set([...GSM_BASIC])
  const ext = new Set([...GSM_EXT])
  let units = 0

  for (const char of message) {
    if (basic.has(char)) {
      units += 1
      continue
    }
    if (ext.has(char)) {
      units += 2
      continue
    }
    return { encoding: 'UCS-2', used: [...message].length, max: 70, single: [...message].length <= 70 }
  }

  return { encoding: 'GSM-7', used: units, max: 160, single: units <= 160 }
}

export function getSMSLengthInfo(message) {
  const text = String(message ?? '')
  return gsmCharUnits(text)
}

export async function sendSMS({ recipient, message, sender }) {
  try {
    return await Backend.SendSMS({ recipient, message, sender })
  } catch (error) {
    throw toError(error, 'Failed to send SMS')
  }
}

export async function listSMSTemplates() {
  try {
    return await Backend.ListSMSTemplates()
  } catch (error) {
    throw toError(error, 'Failed to load SMS templates')
  }
}

export async function createSMSTemplate({ name, body }) {
  try {
    return await Backend.CreateSMSTemplate({ name, body })
  } catch (error) {
    throw toError(error, 'Failed to create template')
  }
}

export async function updateSMSTemplate({ id, name, body }) {
  try {
    return await Backend.UpdateSMSTemplate({ id, name, body })
  } catch (error) {
    throw toError(error, 'Failed to update template')
  }
}

export async function deleteSMSTemplate(id) {
  try {
    return await Backend.DeleteSMSTemplate(id)
  } catch (error) {
    throw toError(error, 'Failed to delete template')
  }
}

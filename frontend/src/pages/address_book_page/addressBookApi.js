import * as Backend from '../../../wailsjs/go/main/App'
import { toError } from '../../errorUtils'

export async function listContacts() {
  try {
    return await Backend.ListContacts()
  } catch (error) {
    throw toError(error, 'Failed to load contacts')
  }
}

export async function importVCF(content) {
  try {
    return await Backend.ImportVCF({ content })
  } catch (error) {
    throw toError(error, 'Failed to import VCF')
  }
}

export async function exportVCF() {
  try {
    return await Backend.ExportVCF()
  } catch (error) {
    throw toError(error, 'Failed to export VCF')
  }
}

export async function createContact(payload) {
  try {
    return await Backend.CreateContact(payload)
  } catch (error) {
    throw toError(error, 'Failed to create contact')
  }
}

export async function updateContact(payload) {
  try {
    return await Backend.UpdateContact(payload)
  } catch (error) {
    throw toError(error, 'Failed to update contact')
  }
}

export async function deleteContact(id) {
  try {
    return await Backend.DeleteContact(id)
  } catch (error) {
    throw toError(error, 'Failed to delete contact')
  }
}

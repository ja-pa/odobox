import * as Backend from '../wailsjs/go/main/App'
import { toError } from './errorUtils'

export async function fetchOdorikBalance() {
  try {
    return await Backend.GetOdorikBalance()
  } catch (error) {
    throw toError(error, 'Failed to load Odorik balance')
  }
}

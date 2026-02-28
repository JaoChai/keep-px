import { test as base } from '@playwright/test'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

export const test = base.extend({
  storageState: path.resolve(__dirname, '../.auth/user.json'),
})

export { expect } from '@playwright/test'

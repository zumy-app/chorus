import { beforeEach, describe, expect, it } from 'vitest'
import {
  clearCryptoStateForTests,
  decryptMessage,
  encryptMessage,
  ensureDeviceKeys,
  generateChatKey,
  getStoredChatKey,
  unwrapChatKey,
  wrapChatKeyForDevice,
} from './index'

describe('E2EE crypto helpers', () => {
  beforeEach(async () => {
    await clearCryptoStateForTests()
  })

  it('generates stable local device key material for a user', async () => {
    const first = await ensureDeviceKeys('user-1')
    const second = await ensureDeviceKeys('user-1')

    expect(first.deviceId).toBe(second.deviceId)
    expect(first.identityPublicKey).toBeTruthy()
    expect(first.signedPreKey).toBeTruthy()
  })

  it('wraps and unwraps a chat key for a device', async () => {
    const device = await ensureDeviceKeys('user-1')
    const chatKey = await generateChatKey('chat-1')

    const envelope = await wrapChatKeyForDevice({
      chatId: 'chat-1',
      userId: 'user-1',
      device,
      chatKey,
    })
    await clearCryptoStateForTests({ keepDeviceKeys: true })

    const unwrapped = await unwrapChatKey('chat-1', envelope)
    expect(unwrapped).toBeDefined()
    expect(await getStoredChatKey('chat-1')).toBeDefined()
  })

  it('encrypts and decrypts message text without preserving plaintext in the payload', async () => {
    await generateChatKey('chat-1')

    const encrypted = await encryptMessage('chat-1', 'hello encrypted world')
    expect(encrypted.ciphertext).toBeTruthy()
    expect(JSON.stringify(encrypted)).not.toContain('hello encrypted world')

    const decrypted = await decryptMessage({
      id: 'msg-1',
      chatId: 'chat-1',
      senderId: 'user-1',
      ciphertext: encrypted.ciphertext,
      nonce: encrypted.nonce,
      algorithm: encrypted.algorithm,
      encryptionVersion: encrypted.encryptionVersion,
      senderDeviceId: encrypted.senderDeviceId,
      text: '',
      deliveryStatus: 'sent',
      timestamp: new Date().toISOString(),
    })

    expect(decrypted.text).toBe('hello encrypted world')
    expect(decrypted.decryptionError).toBeUndefined()
  })

  it('returns a decryption failure state when the local chat key is missing', async () => {
    const encrypted = await encryptMessage('chat-1', 'lost key text')
    await clearCryptoStateForTests()

    const decrypted = await decryptMessage({
      id: 'msg-1',
      chatId: 'chat-1',
      senderId: 'user-1',
      ciphertext: encrypted.ciphertext,
      nonce: encrypted.nonce,
      algorithm: encrypted.algorithm,
      encryptionVersion: encrypted.encryptionVersion,
      senderDeviceId: encrypted.senderDeviceId,
      text: '',
      deliveryStatus: 'sent',
      timestamp: new Date().toISOString(),
    })

    expect(decrypted.text).toBe('')
    expect(decrypted.decryptionError).toContain('Cannot decrypt')
  })
})

import type { EncryptedRecipientKey, Message } from '../../types'

const DEVICE_KEY_PREFIX = 'chorus:e2ee:device:'
const MESSAGE_ALGORITHM = 'AES-GCM'
const KEY_WRAP_ALGORITHM = 'ECDH-P256-AES-GCM'
const ENCRYPTION_VERSION = 1
type StrictUint8Array = Uint8Array & { buffer: ArrayBuffer }

export interface LocalDeviceKeys {
  deviceId: string
  deviceName: string
  deviceType: 'web'
  identityPublicKey: string
  signedPreKey: string
  signedPreKeySignature: string
  oneTimePreKeys: string[]
  privateKey: CryptoKey
}

interface WrapChatKeyArgs {
  chatId: string
  userId: string
  device: Pick<LocalDeviceKeys, 'deviceId' | 'signedPreKey'>
  chatKey: CryptoKey
}

const memoryDeviceKeys = new Map<string, LocalDeviceKeys>()
const memoryChatKeys = new Map<string, CryptoKey>()
let lastDeviceID = ''

export async function ensureDeviceKeys(userId: string): Promise<LocalDeviceKeys> {
  const existing = memoryDeviceKeys.get(userId)
  if (existing) {
    lastDeviceID = existing.deviceId
    return existing
  }

  const storedDeviceID = globalThis.localStorage?.getItem(`${DEVICE_KEY_PREFIX}${userId}:id`)
  const keyPair = await crypto.subtle.generateKey(
    { name: 'ECDH', namedCurve: 'P-256' },
    false,
    ['deriveBits']
  )
  const signedPreKey = await exportPublicKey(keyPair.publicKey)
  const device: LocalDeviceKeys = {
    deviceId: storedDeviceID || crypto.randomUUID(),
    deviceName: navigator.userAgent || 'Chorus Web',
    deviceType: 'web',
    identityPublicKey: signedPreKey,
    signedPreKey,
    signedPreKeySignature: await sha256Base64(signedPreKey),
    oneTimePreKeys: [],
    privateKey: keyPair.privateKey,
  }

  memoryDeviceKeys.set(userId, device)
  lastDeviceID = device.deviceId
  globalThis.localStorage?.setItem(`${DEVICE_KEY_PREFIX}${userId}:id`, device.deviceId)
  return device
}

export async function generateChatKey(chatId: string): Promise<CryptoKey> {
  const existing = memoryChatKeys.get(chatId)
  if (existing) return existing

  const key = await crypto.subtle.generateKey(
    { name: MESSAGE_ALGORITHM, length: 256 },
    true,
    ['encrypt', 'decrypt']
  )
  memoryChatKeys.set(chatId, key)
  return key
}

export async function getStoredChatKey(chatId: string): Promise<CryptoKey | undefined> {
  return memoryChatKeys.get(chatId)
}

export async function storeChatKey(chatId: string, chatKey: CryptoKey): Promise<void> {
  memoryChatKeys.set(chatId, chatKey)
}

export async function wrapChatKeyForDevice({
  chatId,
  userId,
  device,
  chatKey,
}: WrapChatKeyArgs): Promise<EncryptedRecipientKey> {
  const recipientPublicKey = await importPublicKey(device.signedPreKey)
  const ephemeral = await crypto.subtle.generateKey(
    { name: 'ECDH', namedCurve: 'P-256' },
    true,
    ['deriveBits']
  )
  const wrappingKey = await deriveWrappingKey(ephemeral.privateKey, recipientPublicKey)
  const nonce = crypto.getRandomValues(new Uint8Array(12))
  const rawChatKey = await crypto.subtle.exportKey('raw', chatKey)
  const ciphertext = await crypto.subtle.encrypt({ name: MESSAGE_ALGORITHM, iv: nonce }, wrappingKey, rawChatKey)

  return {
    chatId,
    userId,
    deviceId: device.deviceId,
    algorithm: KEY_WRAP_ALGORITHM,
    nonce: bytesToBase64(nonce),
    ciphertext: bytesToBase64(ciphertext),
    ephemeralPublicKey: await exportPublicKey(ephemeral.publicKey),
  }
}

export async function unwrapChatKey(chatId: string, envelope: EncryptedRecipientKey): Promise<CryptoKey> {
  const localDevice = [...memoryDeviceKeys.values()].find((device) => device.deviceId === envelope.deviceId)
  if (!localDevice) {
    throw new Error('Cannot decrypt on this device: missing device private key')
  }
  if (!envelope.ephemeralPublicKey) {
    throw new Error('Cannot decrypt on this device: missing sender key material')
  }

  const ephemeralPublicKey = await importPublicKey(envelope.ephemeralPublicKey)
  const wrappingKey = await deriveWrappingKey(localDevice.privateKey, ephemeralPublicKey)
  const rawChatKey = await crypto.subtle.decrypt(
    { name: MESSAGE_ALGORITHM, iv: base64ToBytes(envelope.nonce) },
    wrappingKey,
    base64ToBytes(envelope.ciphertext)
  )
  const chatKey = await crypto.subtle.importKey('raw', rawChatKey, { name: MESSAGE_ALGORITHM }, true, ['encrypt', 'decrypt'])
  memoryChatKeys.set(chatId, chatKey)
  return chatKey
}

export async function encryptMessage(chatId: string, text: string) {
  const chatKey = await generateChatKey(chatId)
  const nonce = crypto.getRandomValues(new Uint8Array(12))
  const encoded = new TextEncoder().encode(text)
  const ciphertext = await crypto.subtle.encrypt({ name: MESSAGE_ALGORITHM, iv: nonce }, chatKey, encoded)

  return {
    ciphertext: bytesToBase64(ciphertext),
    nonce: bytesToBase64(nonce),
    algorithm: MESSAGE_ALGORITHM,
    encryptionVersion: ENCRYPTION_VERSION,
    senderDeviceId: lastDeviceID || 'local-device',
  }
}

export async function decryptMessage(message: Message): Promise<Message> {
  if (!message.ciphertext) return message

  const chatKey = memoryChatKeys.get(message.chatId)
  if (!chatKey) {
    return { ...message, text: '', decryptionError: 'Cannot decrypt on this device: missing chat key' }
  }

  try {
    const plaintext = await crypto.subtle.decrypt(
      { name: MESSAGE_ALGORITHM, iv: base64ToBytes(message.nonce || '') },
      chatKey,
      base64ToBytes(message.ciphertext)
    )
    return { ...message, text: new TextDecoder().decode(plaintext), decryptionError: undefined }
  } catch {
    return { ...message, text: '', decryptionError: 'Cannot decrypt on this device: decryption failed' }
  }
}

export async function clearCryptoStateForTests(options: { keepDeviceKeys?: boolean } = {}) {
  if (!options.keepDeviceKeys) {
    memoryDeviceKeys.clear()
    lastDeviceID = ''
  }
  memoryChatKeys.clear()
  Object.keys(globalThis.localStorage || {})
    .filter((key) => key.startsWith(DEVICE_KEY_PREFIX))
    .forEach((key) => globalThis.localStorage?.removeItem(key))
}

async function deriveWrappingKey(privateKey: CryptoKey, publicKey: CryptoKey): Promise<CryptoKey> {
  const bits = await crypto.subtle.deriveBits({ name: 'ECDH', public: publicKey }, privateKey, 256)
  const digest = await crypto.subtle.digest('SHA-256', bits)
  return crypto.subtle.importKey('raw', digest, { name: MESSAGE_ALGORITHM }, false, ['encrypt', 'decrypt'])
}

async function exportPublicKey(key: CryptoKey): Promise<string> {
  return bytesToBase64(await crypto.subtle.exportKey('spki', key))
}

async function importPublicKey(encoded: string): Promise<CryptoKey> {
  return crypto.subtle.importKey(
    'spki',
    base64ToBytes(encoded),
    { name: 'ECDH', namedCurve: 'P-256' },
    false,
    []
  )
}

async function sha256Base64(value: string): Promise<string> {
  return bytesToBase64(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(value)))
}

function bytesToBase64(value: ArrayBuffer | Uint8Array): string {
  const bytes = value instanceof Uint8Array ? value : new Uint8Array(value)
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return btoa(binary)
}

function base64ToBytes(value: string): StrictUint8Array {
  const binary = atob(value)
  const bytes = new Uint8Array(new ArrayBuffer(binary.length)) as StrictUint8Array
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return bytes
}

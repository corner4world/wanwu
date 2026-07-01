import service from '@/utils/request';
import { USER_API } from '@/utils/requestConstants';

// 获取 RSA 公钥 + Challenge
async function getRSAPublicKey() {
  const res = await service({
    url: `${USER_API}/base/rsa/public-key`,
    method: 'get',
  });
  return res.data;
}

// PEM → CryptoKey
async function importPublicKey(pem) {
  const b64 = pem
    .replaceAll('-----BEGIN PUBLIC KEY-----', '')
    .replaceAll('-----END PUBLIC KEY-----', '')
    .replaceAll('\n', '');
  const binary = Uint8Array.from(atob(b64), c => c.codePointAt(0));
  return await crypto.subtle.importKey(
    'spki',
    binary,
    { name: 'RSA-OAEP', hash: 'SHA-256' },
    false,
    ['encrypt'],
  );
}

/**
 * RSA-OAEP-SHA256 加密密码
 * @param {string} password - 明文密码
 * @param {object} [keyMaterial] - 可选，复用已获取的公钥+challenge
 * @param {string} keyMaterial.publicKey
 * @param {string} keyMaterial.challenge
 * @returns {Promise<{cipher: string, keyId: string}>}
 */
export async function rsaEncrypt(password, keyMaterial) {
  let keyId, publicKey, challenge;
  if (keyMaterial) {
    ({ keyId, publicKey, challenge } = keyMaterial);
  } else {
    const res = await getRSAPublicKey();
    keyId = res.keyId;
    publicKey = res.publicKey;
    challenge = res.challenge;
  }
  const plaintext = JSON.stringify({ password, challenge });
  const pubKey = await importPublicKey(publicKey);
  const encrypted = await crypto.subtle.encrypt(
    { name: 'RSA-OAEP' },
    pubKey,
    new TextEncoder().encode(plaintext),
  );
  const cipher = btoa(String.fromCodePoint(...new Uint8Array(encrypted)));
  return { cipher, keyId };
}

/**
 * 获取 RSA 公钥 + Challenge（供双 cipher 场景复用）
 * @returns {Promise<{keyId: string, publicKey: string, challenge: string}>}
 */
export async function getRSAKeyMaterial() {
  return await getRSAPublicKey();
}

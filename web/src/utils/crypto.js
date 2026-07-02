import service from '@/utils/request';
import { USER_API } from '@/utils/requestConstants';
import forge from 'node-forge';

// 获取 RSA 公钥 + Challenge
async function getRSAPublicKey() {
  const res = await service({
    url: `${USER_API}/base/rsa/public-key`,
    method: 'get',
  });
  return res.data;
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
  const pubKey = forge.pki.publicKeyFromPem(publicKey);
  const encrypted = pubKey.encrypt(plaintext, 'RSA-OAEP', {
    md: forge.md.sha256.create(),
  });
  const cipher = forge.util.encode64(encrypted);
  return { cipher, keyId };
}

/**
 * 获取 RSA 公钥 + Challenge（供双 cipher 场景复用）
 * @returns {Promise<{keyId: string, publicKey: string, challenge: string}>}
 */
export async function getRSAKeyMaterial() {
  return await getRSAPublicKey();
}

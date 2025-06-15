import bcrypt from 'bcrypt';
import jwt from "jsonwebtoken";
import client from '../Database/config.js';
import { v4 as uuidv4 } from "uuid";
import os from 'os';

/**
 * Encrypts a password using bcrypt.
 * @param {string} password - The password to encrypt.
 * @returns {Promise<string>} - The encrypted password.
 */
export const cryptPassword = (password) => {
  const saltRounds = 10;
  //const hashedPassword = await bcrypt.hash(password, saltRounds);
  //return hashedPassword;
  return bcrypt.hashSync(password, saltRounds);
};

/**
 * Decrypts a password using bcrypt.
 * @param {string} password - The password to decrypt.
 * @param {string} hash - The hash to compare against.
 * @returns {Promise<boolean>} - Whether the password matches the hash.
 */
export const decryptPassword = (password, hash) => {
  //const match = await bcrypt.compare(password, hash);
  //return match;
  return bcrypt.compareSync(password, hash);
};

export const checkToken = (token, JWT_SECRET) => {
  if( jwt.verify(token, JWT_SECRET)){
      return true;
  } else {    
      return false;
  }
};

export const generateUserUid = () => {
  return uuidv4(); // Generates a random UUID, e.g., '1b9d67b2-ad0b-426c-8472-a72a1e0501a9'
}

// ✅ Get local IP dynamically
export const getLocalIp = () => {
  const interfaces = os.networkInterfaces();
  for (const name of Object.keys(interfaces)) {
    for (const iface of interfaces[name]) {
      if (iface.family === 'IPv4' && !iface.internal) {
        return iface.address;
      }
    }
  }
  return '127.0.0.1'; // fallback
}

export async function getUserId(username) {

  const userId = (await client.query(
    "SELECT uuid FROM users WHERE username = $1",
    [username]
  )).rows[0].uuid;

  return userId;
};

const LOGIN_TYPES = Object.freeze({
  EMAIL: 'email',
  SOCIAL_GOOGLE: 'social_google',
  LDAP: 'ldap'
});

export const generateJWT = (uuid) =>{
  // Generate JWT token
  const jwttoken = jwt.sign({ userId: uuid }, process.env.JWT_SECRET, {
    expiresIn: "1h",
  });

  return jwttoken;
}

export { LOGIN_TYPES };
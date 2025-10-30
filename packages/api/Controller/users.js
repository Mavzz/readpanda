import client from '../database/config.js';
import { cryptPassword, decryptPassword, generateUserUid, LOGIN_TYPES, generateTokens, storeRefreshToken, checkToken } from '../utilities/helper.js';
import CryptoJS from "crypto-js";
import dotenv from 'dotenv';
import jwt from "jsonwebtoken";
import { OAuth2Client } from "google-auth-library";


dotenv.config({ path: './.env.local' });

const OAuthclient = new OAuth2Client(process.env.GOOGLE_CLIENT_ID);

/**
 * Get all users from the database.
 * @param {Object} req - The request object.
 * @param {Object} res - The response object.
 */
export const getUsers = async (req, res) => {
  try {
    const result = await client.query("SELECT * FROM users");
    res.status(200).json(result.rows);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
};

/**
 * Create a new user in the database.
 * @param {Object} req - The request object.
 * @param {Object} res - The response object.
 */
export const createUser = async (req, res) => {
  try {

    await client.query('BEGIN');

    const { username, password, email } = req.body;
    // Check if user already exists
    const userCheck = await client.query(
      "SELECT * FROM users WHERE username = $1 OR email = $2",
      [username, email]
    );

    if (userCheck.rows.length > 0) {
      return res.status(409).json({ error: "User already exists" });
    }

    // Decrypt the password
    const bytes = CryptoJS.AES.decrypt(password, process.env.CRYPTO_SECRET);
    const decryptedPassword = bytes.toString(CryptoJS.enc.Utf8);

    // Insert new user
    const hashedPassword = cryptPassword(decryptedPassword);

    // Generate a unique user ID
    const newUserUid = generateUserUid();

    const result = await client.query(
      "INSERT INTO users (username, password, email, isactive, login_type, uuid) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *",
      [username, hashedPassword, email, true, LOGIN_TYPES.EMAIL, newUserUid]
    );

    // Insert user preferences and get the inserted preferences
    const preferencesData = await initializeUserPreferences(newUserUid);

    // Generate tokens
    const { accessToken, refreshToken } = generateTokens(newUserUid);
    await storeRefreshToken(newUserUid, refreshToken);

    await client.query('COMMIT');

    res.status(201).json({
      "username": result.rows[0].username,
      "email": result.rows[0].email,
      "preferences": preferencesData,
      "accessToken": accessToken,
      "refreshToken": refreshToken
    });
  } catch (err) {
    await client.query('ROLLBACK');
    res.status(500).json({ error: err.message });
  }
};

/**
 * Log in a user and generate a JWT token.
 * @param {Object} req - The request object.
 * @param {Object} res - The response object.
 */
export const loginUser = async (req, res) => {
  const { username, password } = req.body;
  try {

    // Decrypt the password
    const bytes = CryptoJS.AES.decrypt(password, process.env.CRYPTO_SECRET);
    const decryptedPassword = bytes.toString(CryptoJS.enc.Utf8);

    // Check if user exists
    const userCheck = await client.query(
      "SELECT * FROM users WHERE username = $1",
      [username]
    );

    if (userCheck.rows.length === 0) {
      return res.status(401).json({ error: "Invalid username or password" });
    }

    const user = userCheck.rows[0];

    // Verify password
    const isPasswordValid = decryptPassword(decryptedPassword, user.password);

    if (!isPasswordValid) {
      return res.status(401).json({ error: "Invalid username or password" });
    }

    // Generate JWT token
    if (!process.env.JWT_SECRET) {
      return res.status(500).json({ error: "JWT_SECRET is not defined" });
    }
    if (!process.env.JWT_REFRESH_SECRET) {
      return res.status(500).json({ error: "JWT_REFRESH_SECRET is not defined" });
    }

    // Generate tokens
    const { accessToken, refreshToken } = generateTokens(user.uuid);
    await storeRefreshToken(user.uuid, refreshToken);

    res.status(200).json({
      "accessToken": accessToken,
      "refreshToken": refreshToken,
      "username": user.username,
      "email": user.email
    });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
};

/**
 * Login user using google SSO and generate a JWT token.
 * @param {Object} req - The request object.
 * @param {Object} res - The response object.
 */

export const googleAuth = async (req, res) => {

  try {

    const authHeader = req.header("Authorization");
    const token = authHeader && authHeader.split(" ")[1];

    const ticket = await OAuthclient.verifyIdTokenAsync({
      idToken: token,
      audience: process.env.GOOGLE_CLIENT_ID, // This should match the client ID used on frontend
    });
    const payload = ticket.getPayload();
    const { email, name, sub: id, picture } = payload; // sub is the unique identifier for the user

    // Check if user already exists in the database
    const userCheck = await client.query(
      "SELECT * FROM users WHERE email = $1",
      [email]
    );

    let user;
    if (userCheck.rows.length > 0) {

      const { accessToken, refreshToken } = generateTokens(userCheck.rows[0].uuid);
      await storeRefreshToken(userCheck.rows[0].uuid, refreshToken);

      return res.status(200).json({
        "username": userCheck.rows[0].username,
        "picture": picture,
        "accessToken": accessToken,
        "refreshToken": refreshToken
      });

    } else {
      // Generate a unique user ID
      const newUserUid = generateUserUid();
      // Create a new user if not exists
      const result = await client.query(
        "INSERT INTO users (username, email, isactive, login_type, uuid, google_sub) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *",
        [name, email, true, LOGIN_TYPES.SOCIAL_GOOGLE, newUserUid, id]
      );
      user = result.rows[0];

      // Insert user preferences and get the inserted preferences
      await initializeUserPreferences(newUserUid);
      // Generate tokens
      const { accessToken, refreshToken } = generateTokens(user.uuid);
      await storeRefreshToken(user.uuid, refreshToken);

      const username = user.username;

      // Return the user data and token
      res.status(201).json({
        "username": username,
        "picture": picture,
        "accessToken": accessToken,
        "refreshToken": refreshToken
      });
    }

  } catch (err) {
    res.status(500).json({ error: err.message });
  }
};

/**
 * Login user using google SSO and generate a JWT token.
 * @param {Object} req - The request object.
 * @param {Object} res - The response object.
 */

// New refresh token endpoint
export const refreshAccessToken = async (req, res) => {
  try {
    const { refreshToken } = req.body;

    if (!refreshToken) {
      return res.status(401).json({ error: 'Refresh token required' });
    }

    // Verify refresh token
    const decoded = jwt.verify(refreshToken, process.env.JWT_REFRESH_SECRET);

    if (decoded.type !== 'refresh') {
      return res.status(401).json({ error: 'Invalid token type' });
    }

    // Check if refresh token exists in database
    const tokenCheck = await client.query(
      'SELECT * FROM refresh_tokens WHERE user_id = $1 AND token = $2 AND expires_at > NOW()',
      [decoded.userId, refreshToken]
    );

    if (tokenCheck.rows.length === 0) {
      return res.status(401).json({ error: 'Invalid or expired refresh token' });
    }

    // Generate new tokens
    const { accessToken } = generateTokens(decoded.userId);

    res.status(200).json({
      "accessToken": accessToken,
    });
  } catch (err) {
    if (err.name === 'JsonWebTokenError' || err.name === 'TokenExpiredError') {
      return res.status(401).json({ error: 'Invalid or expired refresh token' });
    }
    res.status(500).json({ error: err.message });
  }
};

/**
 * Login user using google SSO and generate a JWT token.
 * @param {Object} req - The request object.
 * @param {Object} res - The response object.
 */

// Logout user by invalidating the refresh token
export const logoutUser = async (req, res) => {
  try {
    const authHeader = req.header("Authorization");
    const token = authHeader && authHeader.split(" ")[1]; // Bearer <token>
    const { username } = req.query;
    const userId = await getUserId(username);

    if (checkToken(token, process.env.JWT_REFRESH_SECRET)) {
      await client.query('DELETE FROM refresh_tokens WHERE token = $1 AND user_id = $2', [token, userId]);
    }

    res.status(200).json({ message: 'Logged out successfully' });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
};

export const getUserId = async (username) => {
  try {
    const userCheck = await client.query(
      "SELECT * FROM users WHERE username = $1",
      [username]
    );

    if (userCheck.rows.length === 0) {
      throw new Error("User not found");
    }

    return userCheck.rows[0].uuid;
  } catch (err) {
    throw new Error("Error fetching user ID");
  }
};

async function initializeUserPreferences(newUserUid) {
  const preferencesData = {};

  for (const preference of (
    await client.query("SELECT * FROM preferences order by id")
  ).rows) {
    if (!preferencesData[preference.genre]) {
      preferencesData[preference.genre] = [];
    }
    preferencesData[preference.genre].push({
      preference_id: preference.id,
      preference_subgenre: preference.subgenre,
      preference_value: false
    });
  }

  const preferences = await client.query(
    "INSERT INTO user_preferences (user_id, preferences) VALUES ($1, $2)",
    [newUserUid, preferencesData]
  );
  return preferencesData;
}

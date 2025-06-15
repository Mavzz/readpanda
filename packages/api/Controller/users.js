import client from '../Database/config.js';
import { cryptPassword, decryptPassword, generateUserUid, LOGIN_TYPES, generateJWT } from '../Utilities/helper.js';
import CryptoJS from "crypto-js";
import dotenv from 'dotenv';
import jwt from "jsonwebtoken";
import { OAuth2Client } from "google-auth-library";

dotenv.config();

const OAuthclient = new OAuth2Client(process.env.GOOGLE_CLIENT_ID);

// Initialize Firebase Admin SDK
/*admin.initializeApp({
  credential: admin.credential.cert(serviceAccount),
});*/

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
    res.status(201).json({
      "username": result.rows[0].username,
      "email": result.rows[0].email,
      "token": jwt.sign({ userId: result.rows[0].uuid }, process.env.JWT_SECRET, {
        expiresIn: "1h",
      }),
    });
  } catch (err) {
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
    const token = jwt.sign({ userId: user.id }, process.env.JWT_SECRET, {
      expiresIn: "1h",
    });

    res.status(200).json({ token });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
};

/**
 * Log in a user and generate a JWT token.
 * @param {Object} req - The request object.
 * @param {Object} res - The response object.
 */

export const googleAuth = async (req, res) => {

  try {
    
    const { token } = req.body;
    
    const ticket = await OAuthclient.verifyIdTokenAsync({
    idToken: token,
    audience: process.env.GOOGLE_CLIENT_ID, // This should match the client ID used on frontend
  });
    const payload = ticket.getPayload();
    const { email, name, sub: id } = payload; // sub is the unique identifier for the user

    // Check if user already exists in the database
    const userCheck = await client.query(
      "SELECT * FROM users WHERE email = $1",
      [email]
    );

    let user;
    // Generate a unique user ID
    const newUserUid = generateUserUid();
    if (userCheck.rows.length > 0) {
      
      return res.status(200).json({ username: userCheck.rows[0].username, token: generateJWT(userCheck.rows[0].uuid) });

    } else {
      // Create a new user if not exists
      const result = await client.query(
        "INSERT INTO users (username, email, isactive, login_type, uuid, google_sub) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *",
        [name, email, true, LOGIN_TYPES.SOCIAL_GOOGLE, newUserUid, id]
      );
      user = result.rows[0];
    }

    const username = user.username;

    // Return the user data and token
    res.status(201).json({ username: username, token: generateJWT(user.uuid) });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
};
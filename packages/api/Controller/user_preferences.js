import client from "../Database/config.js";
import dotenv from "dotenv";
import { checkToken, getUserId } from "../Utilities/helper.js";

dotenv.config();

/**
 * Get all users from the database.
 * @param {Object} req - The request object.
 * @param {Object} res - The response object.
 */
export const getUserPreferences = async (req, res) => {
  try {
    const authHeader = req.header("authorization");
    const token = authHeader && authHeader.split(" ")[1]; // Bearer <token>
    const { username } = req.query;

    //console.log(token);
    console.log(username);

    if (checkToken(token, process.env.JWT_SECRET)) {
      const user_preferenceCheck = await client.query(
        "SELECT user_id FROM user_preferences WHERE user_id = (SELECT uuid FROM users WHERE username = $1)",
        [username]
      );

      let userId;
      let preferences;
      if (user_preferenceCheck.rows.length === 0) {
        userId = (
          await client.query("SELECT uuid FROM users WHERE username = $1", [
            username,
          ])
        ).rows[0].uuid;

        // Insert user preferences and get the inserted preferences
        const preferencesData = {}
        
        /*preferencesArray.push({
            preference_id: preference.id,
            preference_subgenre: preference.subgenre,
            preference_genre: preference.genre,
            preference_value: false,
          });*/

        for (const preference of (
          await client.query("SELECT * FROM preferences order by id")
        ).rows) {
          if (!preferencesData[preference.genre]) {
            preferencesData[preference.genre] = [];
          }
          preferencesData[preference.genre].push({
            preference_id: preference.id,
            preference_subgenre: preference.subgenre,
            preference_value: false});
        }

        //console.log(preferencesArray);

        preferences = await client.query(
          "INSERT INTO user_preferences (user_id, preferences) VALUES ($1, $2)",
          [userId, preferencesData]
        );
        res.status(201).json(preferencesData);
        console.log("User preferences inserted successfully");
      } else {
        userId = user_preferenceCheck.rows[0].user_id;
        preferences = await client.query(
          "SELECT preferences FROM user_preferences WHERE user_id = $1",
          [userId]
        );
        console.log("User preferences retrieved successfully ");
        res.status(200).json(preferences.rows[0].preferences);
      }
    } else {
      res.status(401).json({ error: "Unauthorized" });
    }
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
};

export const updateUserPreferences = async (req, res) => {
  try {
    const authHeader = req.header("authorization");
    let token = authHeader && authHeader.split(" ")[1]; // Bearer <token>
    token = token.replace(/^"|"$/g, "");
    //console.log(token);
    const { username } = req.query;
    let { preferences } = req.body;

    if (checkToken(token, process.env.JWT_SECRET)) {
      const userId = await getUserId(username);
      const updatepreferences = await client.query(
        `UPDATE user_preferences 
        SET preferences = $1, updated_at = NOW() 
        WHERE user_id = $2`,
        [preferences, userId]
      );

      res.status(200).json({"message": "User preferences updated successfully"});
    } else {
      res.status(401).json({ error: "Unauthorized" });
    }
  } catch (err) {
    console.log(err);
    res.status(500).json({ error: err.message });
  }
};

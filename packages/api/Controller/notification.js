import client from '../database/config.js';
import dotenv from 'dotenv';
import { checkToken, getUserId } from "../utilities/helper.js";

dotenv.config({path: './.env.local'});


export const getUserNotifications = async (req, res) => {
    try {
        const authHeader = req.header("authorization");
        if (!authHeader) {
            return res.status(401).json({ error: "Unauthorized" });
        }
        const token = authHeader.split(" ")[1];
        const { username } = req.query;
        
        if (checkToken(token, process.env.JWT_SECRET)) {
            const userId = await getUserId(username);

            const user_notifications = await client.query(
                `SELECT * FROM notifications WHERE user_id = $1 ORDER BY created_at DESC`,
                [userId]
            );

            if (user_notifications.rows.length > 0) {
                res.json(user_notifications.rows);
            } else {
                res.status(404).json({ message: "No notifications found" });
            }

        }

    } catch (error) {
        res.status(500).json({ error: error.message });
    }
};

export const getUnreadNotificationCount = async (req, res) => {
    try {
        const authHeader = req.header("authorization");
        if (!authHeader) {
            return res.status(401).json({ error: "Unauthorized" });
        }
        const token = authHeader.split(" ")[1];
        const { username } = req.query;
        
        if (checkToken(token, process.env.JWT_SECRET)) {
            const userId = await getUserId(username);

            const unread_count_result = await client.query(
                `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`,
                [userId]
            );

            const unread_count = parseInt(unread_count_result.rows[0].count, 10);

            res.json({ unread_count });

        }

    } catch (error) {
        res.status(500).json({ error: error.message });
    }
};

// Other existing functions (if any) can go here
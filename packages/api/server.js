import express from 'express';
import bodyParser from 'body-parser';
import routes from './Routes/routing.js';
import dotenv from 'dotenv';
import client from './database/config.js';
import fs from 'fs';
import http from 'http';
import { getLocalIp } from './utilities/helper.js';


dotenv.config({path: './.env.local'});

const app = express();

// Middleware for parsing request body
app.use(bodyParser.json());

app.use((req, res, next) =>{
  res.setHeader("access-control-allow-origin", "*");
  res.setHeader("access-control-allow-methods", "GET, POST, PUT, PATCH, DELETE");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type, Authorization"); // Corrected header for allowed request headers
  res.setHeader("access-control-expose-headers", "Content-Type, Authorization");
  next();
})

app.use((req, res, next) => {
  console.log(`Incoming ${req.method} request to ${req.url}`);
  next();
});

// Connect to the database
client
  .connect()
  .then(() => console.log("Connected to PostgreSQL"))
  .catch((err) => console.error("Connection error", err.stack));

/*app.use((req, res, next) => {
  const authHeader = req.header("authorization");
  const token = authHeader && authHeader.split(" ")[1];
  if (token) {
    console.log("Authorization token found");
    next();
  } else {
    console.error("No authorization token found");
    res.status(401).json({ error: "Unauthorized" });
  }
  
});*/

// Routes for API endpoints
app.use("/", routes);

// 🛡️ Read cert files
const sslOptions = {
  key: fs.readFileSync('key.pem'),
  cert: fs.readFileSync('cert.pem'),
};

const localIp = "0.0.0.0"

// Server listening on port 3000 for incoming requests
const port = process.env.PORT || 3000;

http.createServer(sslOptions, app).listen(port, localIp, () => {
  console.log(`App running on ${localIp} HTTPS and port ${port}...`);
});
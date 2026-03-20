import express from 'express';
import bodyParser from 'body-parser';
import routes from './Routes/routing.js';
import dotenv from 'dotenv';
import client from './database/config.js';
import fs from 'fs';
import http from 'http';
import https from 'https';
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

const localIp = "0.0.0.0"
const port = process.env.PORT || 3000;

// Start HTTP server (use a reverse proxy like nginx for HTTPS in production)
http.createServer(app).listen(port, localIp, () => {
  console.log(`App running on ${localIp} HTTP port ${port}...`);
});

// Optionally start HTTPS if cert files exist
try {
  const sslOptions = {
    key: fs.readFileSync('key.pem'),
    cert: fs.readFileSync('cert.pem'),
  };
  const httpsPort = process.env.HTTPS_PORT || 3443;
  https.createServer(sslOptions, app).listen(httpsPort, localIp, () => {
    console.log(`App running on ${localIp} HTTPS port ${httpsPort}...`);
  });
} catch {
  console.log('No SSL certs found (key.pem/cert.pem) — HTTPS server not started');
}
import CryptoJS  from "crypto-js";
import { SECRET_KEY } from "@env";
import dotenv from 'dotenv';
dotenv.config();

// Encrypt the password
const encryptedPassword = (password) => {
  return CryptoJS.AES.encrypt(password, SECRET_KEY).toString();
};

const getBackendUrl = async(path = "") => {

  //const apiUrl = `https://${Constants.expoConfig.hostUri.split(':')[0]}:3000`;

  let backendUrl;
    try {
      
      const ip = "192.168.0.104" //await Network.getIpAddressAsync();
      const port = 3000; // your backend port
      backendUrl = `http://${ip}:${port}${path}`;

    return backendUrl;

  } catch {
    console.warn("⚠️ Failed to get local IP, falling back to localhost");
    return `http://localhost:3000${path}`;
  }
};

const SignUpType = {
  Email : "Email",
  Google : "Google",
  Facebook : "Facebook",
  Other : "Other"
}

/*const googleSignUpLogin = async () => {
  let status;
  let response;
  // Attempt to sign in with Google
  const token = await GoogleSignInModule.signIn();
  console.log("Google ID Token:", token);

  if (token) {
    ({ status, response } = await UsePOST(await getBackendUrl("/auth/google"), {
      token,
    }));

    console.log("Signup Response:", response);
  } else {
    console.warn("Login failed", "An error occurred. Please try again.");
  }
  return { status, response };
};*/

export { encryptedPassword, getBackendUrl, SignUpType };

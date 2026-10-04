import CryptoJS  from "crypto-js";

// Encrypt the password
const encryptedPassword = (password) => {
  return CryptoJS.AES.encrypt(password, import.meta.env.VITE_CRYPTO_SECRET).toString();
};

// VITE_BACKEND_URL is the API's full origin, e.g.
// https://readpanda-backend-….run.app (production: https, no port). Without
// it, local dev builds http://<VITE_BACKEND_BASE_URL>:<VITE_BACKEND_PORT>.
const getBackendUrl = async(path = "") => {
  const apiVersion = import.meta.env.VITE_API_VERSION || '/api/v1';
  const origin = import.meta.env.VITE_BACKEND_URL;
  if (origin) return `${origin.replace(/\/+$/, '')}${apiVersion}${path}`;
  const ip = import.meta.env.VITE_BACKEND_BASE_URL || 'localhost';
  const port = import.meta.env.VITE_BACKEND_PORT || 3000;
  return `http://${ip}:${port}${apiVersion}${path}`;
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

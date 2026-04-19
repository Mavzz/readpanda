import CryptoJS  from "crypto-js";

// Encrypt the password
const encryptedPassword = (password) => {
  return CryptoJS.AES.encrypt(password, import.meta.env.VITE_CRYPTO_SECRET).toString();
};

const getBackendUrl = async(path = "") => {
  const ip = import.meta.env.VITE_BACKEND_BASE_URL || 'localhost';
  const port = import.meta.env.VITE_BACKEND_PORT || 3000;
  const apiVersion = import.meta.env.VITE_API_VERSION || '/api/v1';
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

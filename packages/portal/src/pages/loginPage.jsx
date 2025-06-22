import React, { useState } from "react";
import { usePost as UsePOST } from "../services/usePost";
import { getBackendUrl, encryptedPassword } from "../utils/Helper";
import { GoogleLogin } from "@react-oauth/google"; // Import GoogleLogin component
import { useNavigate } from 'react-router-dom'; // Import useNavigate

const LoginPage = ({ setIsLoggedIn }) => {
  const [loading, setLoading] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState(""); // State for error messages
  const navigate = useNavigate(); // Hook for programmatic navigation
  
  // Define a timeout duration (e.g., 10 seconds)
  const LOGIN_TIMEOUT_MS = 10000;

  const handleLogin = async () => {
    // Clear previous errors
    setError("");

    if (!email || !password) {
      setError("Please enter both email and password.");
      console.warn("Login attempt with empty email or password");
      return;
    }

    setLoading(true);
    let timeoutId; // To store the timeout ID

    // Create an AbortController instance for this request
    const controller = new AbortController();
    const signal = controller.signal;

    try {
      // Set a timeout to abort the request
      timeoutId = setTimeout(() => {
        controller.abort();
        setError("Login request timed out. Please try again.");
        setLoading(false); // Reset loading state here as well
      }, LOGIN_TIMEOUT_MS);

      const { status, response } = await UsePOST(
        await getBackendUrl("/auth/login"),
        {
          email,
          password: encryptedPassword(password),
        },
        {}, // Headers can be passed if needed
        signal // Pass the signal to the usePost function
      );

      clearTimeout(timeoutId); // Clear the timeout if request completes before timeout

      if (status === 200 && response.token) {
        // Changed assignment to comparison
        localStorage.setItem("token", response.token);
        localStorage.setItem("username", response.username);
        setIsLoggedIn(true);
        console.log("Login successful with email:", email);
        // Maybe a success toast here
      } else {
        // Handle non-200 but token exists scenario if applicable
        setError(
          response.error ||
            "Login failed: Invalid credentials or server error. Please try again."
        );
      }
    } catch (error) {
      clearTimeout(timeoutId); // Clear the timeout if request completes before timeout

      if (error.name === "AbortError") {
        // This error is handled by the setTimeout callback, so we just log it
        console.warn("Login request aborted due to timeout.");
      } else {
        console.error("Login API error:", error);
        setError(
          "An unexpected error occurred during login. Please check your network connection and try again."
        );
      }
    } finally {
      setLoading(false);
    }
  };

  // New handler for Google Sign-in success
  const handleGoogleSuccess = async (credentialResponse) => {
    setLoading(true);
    setError(""); // Clear any previous errors
    console.log("Google Sign-in successful:", credentialResponse);
    try {
      // Send the Google ID token (credentialResponse.credential) to your backend
      const { status, response } = await UsePOST(
        await getBackendUrl("/auth/google"), // This is your backend endpoint
        { token: credentialResponse.credential }
      );

      console.log("Google Sign-in Response:", response);
      console.log("Google Sign-in Status:", status);
      if (status === 201 || status === 200) {
        // 201 for new user, 200 for existing
        localStorage.setItem("token", response.token);
        localStorage.setItem("username", response.username);
        setIsLoggedIn(true);
        console.log("Google Sign-in successful:", response.username);
        navigate("/dashboard");
      } else {
        setError(
          response.error || "Google Sign-in failed on server. Please try again."
        );
      }
    } catch (error) {
      console.error("Google Sign-in API error:", error);
      setError("An error occurred during Google Sign-in. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  // Handler for Google Sign-in failure
  const handleGoogleFailure = (errorResponse) => {
    console.error("Google Sign-in failed:", errorResponse);
    setError("Google Sign-in was unsuccessful. Please try again.");
    // setLoading(false); // No need to set loading to false here, as it's not set to true for failure
  };

  return (
    <div className="w-screen h-screen bg-gray-100 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-8">
        <div>
          <h1 className="text-center text-4xl font-extrabold text-gray-900">
            ReadPanda
          </h1>
          <h2 className="mt-2 text-center text-2xl font-bold text-indigo-600">
            Writer's Portal
          </h2>
          <p className="mt-2 text-center text-sm text-gray-600">
            Sign in to your account
          </p>
        </div>
        <form className="mt-8 space-y-6">
          {" "}
          {/* Use onSubmit on the form */}
          {/* Display error message if present */}
          {error && (
            <div
              className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative"
              role="alert"
            >
              <strong className="font-bold">Error: </strong>
              <span className="block sm:inline">{error}</span>
            </div>
          )}
          <div className="rounded-md shadow-sm -space-y-px">
            <div>
              <label htmlFor="email-address" className="sr-only">
                Email address
              </label>
              <input
                id="email-address"
                name="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                type="email"
                autoComplete="email"
                required
                className="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-t-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
                placeholder="Email address"
              />
            </div>
            <div>
              <label htmlFor="password" className="sr-only">
                Password
              </label>
              <input
                id="password"
                name="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                type="password"
                autoComplete="current-password"
                required
                className="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-b-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
                placeholder="Password"
              />
            </div>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <input
                id="remember-me"
                name="remember-me"
                type="checkbox"
                className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
              />
              <label
                htmlFor="remember-me"
                className="ml-2 block text-sm text-gray-900"
              >
                {" "}
                Remember me{" "}
              </label>
            </div>

            <div className="text-sm">
              <a
                href="#"
                className="font-medium text-indigo-600 hover:text-indigo-500"
              >
                {" "}
                Forgot your password?{" "}
              </a>
            </div>
          </div>
          <div>
            <button
              type="submit"
              disabled={loading}
              onClick={(e) => {
                e.preventDefault(); // Prevent default form submission
                handleLogin(); // Call the login handler
              }}
              className={`group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 ${
                loading ? "opacity-50 cursor-not-allowed" : ""
              }`}
            >
              {loading ? "Signing In..." : "Sign in"}
            </button>
          </div>
          <div className="mt-6">
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-gray-300" />
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="px-2 bg-gray-100 text-gray-500">
                  Or sign in with
                </span>
              </div>
            </div>
          </div>
          <GoogleLogin
              onSuccess={handleGoogleSuccess}
              onError={handleGoogleFailure}
              theme="filled_blue" // or 'outline'
              size="large" // or 'medium', 'small'
              text-align="center"
              width="360px" // Adjust width as needed
              disabled={loading} // Disable Google button too when any login is in progress
            />
        </form>
      </div>
    </div>
  );
};

export default LoginPage;

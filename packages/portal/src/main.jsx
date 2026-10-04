import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom"; // Import BrowserRouter
import "./index.css";
import App from "./App.jsx";
import { GoogleOAuthProvider } from "@react-oauth/google"; // Import GoogleOAuthProvider
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import ConfirmProvider from "./components/ConfirmProvider.jsx";

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID; // Assuming Vite for environment variables
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // A 4xx won't fix itself on retry; only retry network and server errors.
      retry: (count, err) => !(err?.status >= 400 && err.status < 500) && count < 2,
    },
  },
});

ReactDOM.createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <GoogleOAuthProvider clientId={GOOGLE_CLIENT_ID}>
        <BrowserRouter>
          <ConfirmProvider>
            <App />
          </ConfirmProvider>
        </BrowserRouter>
      </GoogleOAuthProvider>
    </QueryClientProvider>
  </React.StrictMode>
);

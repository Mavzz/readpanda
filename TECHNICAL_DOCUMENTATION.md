# ReadPanda: Technical Overview & Architecture

## 1. Project Overview

ReadPanda is a full-stack web application designed to create a "virtual book club" experience. It allows users to form private or public groups (called "rooms") to read books together. The core feature is the ability to comment on specific parts of a book, with comments being visible only to members of that room. This fosters a shared reading experience, merging social interaction with a digital reading platform.

- **Target User:** Readers who want to discuss books with a select group of friends without public visibility and the spoilers that come with less-structured platforms.
- **Core Problem Solved:** Provides a dedicated, private space for synchronous and asynchronous book discussions, directly tied to the book's content.

---

## 2. System Architecture

ReadPanda is built using a **monorepo architecture**, containing the backend and frontend codebases in a single repository. This simplifies dependency management and cross-application consistency. The architecture follows a classic **client-server model**.

- **Backend (`api`):** A Node.js server that exposes a RESTful API. It handles business logic, user authentication, database interactions, and file storage.
- **Frontend (`portal`):** A single-page application (SPA) built with React. It provides the user interface and consumes the API exposed by the backend.

### 2.1. Project Structure

The repository is organized into two main packages:

```
/
├── packages/
│   ├── api/         # Backend Node.js/Express Application
│   └── portal/      # Frontend React/Vite Application
├── .gitignore
├── package.json     # Root package file (likely for workspace management)
└── README.md
```

---

## 3. Backend Architecture (`packages/api`)

The backend is a modern JavaScript application running on Node.js and the Express framework.

- **Stack:**
  - **Runtime:** Node.js
  - **Framework:** Express.js
  - **Database:** PostgreSQL
  - **File Storage:** Firebase Cloud Storage
  - **Authentication:** JSON Web Tokens (JWT) & Google OAuth
- **Key Dependencies:**
  - `express`: Web server framework.
  - `pg`: PostgreSQL client for database connectivity.
  - `jsonwebtoken`: For creating and verifying JWTs.
  - `bcrypt`: For hashing user passwords.
  - `firebase-admin`: For backend integration with Firebase services (like Storage).
  - `googleapis`: To interact with Google Drive API.
  - `multer`: Middleware for handling `multipart/form-data`, used for file uploads.
  - `dotenv`: For managing environment variables.

### 3.1. Directory Structure (`api`)

```
packages/api/
├── controller/      # Handles request/response logic (the "C" in MVC).
├── database/      # Database connection config (config.js) and SQL queries.
├── routes/        # Defines API endpoints and maps them to controllers.
├── service/       # Contains business logic for complex operations (e.g., file uploads).
├── utilities/     # Helper functions and miscellaneous utilities.
├── server.js      # Main application entry point.
└── package.json
```

### 3.2. API & Routing

- Routing is centralized in `routes/routing.js`, which imports controllers to handle specific endpoints.
- The API likely includes endpoints for user management (`/users`), file operations (`/files`), and application-specific logic.

### 3.3. Authentication

- The system supports both email/password and Google-based authentication.
- **JWT Flow:** Upon successful login, the server generates a JWT and sends it to the client. The client must include this token in the `Authorization` header of subsequent requests to access protected routes.
- **Password Security:** Passwords are not stored in plaintext. They are hashed using `bcrypt` before being saved to the database.

### 3.4. Database

- A **PostgreSQL** relational database is used for data persistence.
- Connection configuration is managed in `database/config.js`.
- Database schema and initial queries can be found in `database/queries.sql`.

### 3.5. File Handling

- **Primary Storage:** E-books and user-uploaded files are stored in **Firebase Cloud Storage**. The `service/firebaseStorageService.js` contains the logic for interacting with Firebase.
- **Secondary Integration:** There is an integration with **Google Drive API** (`controller/GoogleDrive.js`), which may be used for importing books or other file-related features.

---

## 4. Frontend Architecture (`packages/portal`)

The frontend is a modern, responsive Single-Page Application (SPA) built with React.

- **Stack:**
  - **Framework:** React
  - **Build Tool:** Vite
  - **Routing:** React Router
  - **Styling:** Tailwind CSS
  - **Authentication:** Google OAuth Client Library
- **Key Dependencies:**
  - `react` & `react-dom`: Core libraries for building the UI.
  - `react-router-dom`: For client-side routing and navigation.
  - `@react-oauth/google`: Simplifies the Google OAuth 2.0 flow on the client-side.
  - `vite`: Fast and modern build tool for frontend development.
  - `tailwindcss`: A utility-first CSS framework for rapid UI development.

### 4.1. Directory Structure (`portal`)

```
packages/portal/
├── public/          # Static assets (e.g., logos, favicon).
├── src/
│   ├── assets/      # Component-specific assets.
│   ├── components/  # Reusable React components (e.g., buttons, modals, icons).
│   ├── pages/       # Top-level components representing application pages (e.g., LoginPage, DashboardPage).
│   ├── services/    # Logic for making API calls (e.g., custom hooks like useGet, usePost).
│   ├── utils/       # Helper functions for the frontend.
│   ├── App.jsx      # Main application component, contains routing logic.
│   ├── main.jsx     # Application entry point.
│   └── index.css    # Global styles.
├── vite.config.js   # Vite configuration.
└── package.json
```

### 4.2. Architecture & State Management

- **Component-Based:** The UI is built from a hierarchy of reusable React components, promoting modularity and maintainability.
- **Routing:** `react-router-dom` is used to manage navigation between different pages (`/login`, `/dashboard`, `/my-books`, etc.) without full page reloads. The main routing logic is defined in `src/App.jsx`.
- **State Management:** The application primarily uses local component state via React Hooks (`useState`, `useEffect`). For cross-component state, it passes props down the component tree. There is no global state management library (like Redux or Zustand) currently integrated.
- **API Interaction:** Custom hooks in `src/services/` are likely used to encapsulate the logic for fetching data from and posting data to the backend API.

---

## 5. Getting Started Guide

Follow these steps to set up and run the ReadPanda application locally.

### 5.1. Prerequisites

- Node.js (v18 or later recommended)
- npm (or yarn)

### 5.2. Installation

1.  **Clone the repository:**
    ```bash
    git clone <repository-url>
    cd readpanda
    ```
2.  **Install all dependencies:**
    Run the installation command from the root of the monorepo. This will install dependencies for both the `api` and `portal` packages.
    ```bash
    npm install
    ```

### 5.3. Running the Application

You need to run the backend and frontend in separate terminals.

1.  **Run the Backend Server:**
    ```bash
    cd packages/api
    npm start
    ```
    The API server should now be running, typically on `https://localhost:3000`.

2.  **Run the Frontend Portal:**
    ```bash
    cd packages/portal
    npm run dev
    ```
    The React development server will start, and the application will be accessible in your browser, typically at `http://localhost:5173`.

---

## 6. Environment Configuration

Both the backend and frontend require environment variables for configuration (e.g., database credentials, API keys). These are managed in `.env.local` files, which are not committed to version control.

1.  **Backend (`packages/api/.env.local`):**
    Create a file named `.env.local` inside the `packages/api` directory. It should contain the following variables:
    ```
    # Server Configuration
    PORT=3000

    # PostgreSQL Database
    DB_USER=your_db_user
    DB_HOST=localhost
    DB_DATABASE=your_db_name
    DB_PASSWORD=your_db_password
    DB_PORT=5432

    # JWT
    JWT_SECRET=your_super_secret_jwt_key

    # Firebase (copy from your Firebase project settings)
    FIREBASE_TYPE=service_account
    FIREBASE_PROJECT_ID=...
    FIREBASE_PRIVATE_KEY_ID=...
    FIREBASE_PRIVATE_KEY=...
    FIREBASE_CLIENT_EMAIL=...
    # ... and other Firebase service account credentials
    ```

2.  **Frontend (`packages/portal/.env.local`):**
    Create a file named `.env.local` inside the `packages/portal` directory.
    ```
    # The base URL of the backend API
    VITE_API_BASE_URL=https://localhost:3000

    # Google OAuth Client ID (from Google Cloud Console)
    VITE_GOOGLE_CLIENT_ID=your_google_client_id.apps.googleusercontent.com
    ```

package com.readpanda.app;

import android.app.Activity
import android.util.Log
import androidx.credentials.CredentialManager
import androidx.credentials.GetCredentialRequest
import com.facebook.react.bridge.*
import com.google.android.libraries.identity.googleid.GetGoogleIdOption
import com.google.android.libraries.identity.googleid.GoogleIdTokenCredential
import kotlinx.coroutines.*
import kotlin.coroutines.CoroutineContext

class GoogleSignInModule(
    private val reactContext: ReactApplicationContext
) : ReactContextBaseJavaModule(reactContext), CoroutineScope {

    private val job = SupervisorJob()
    override val coroutineContext: CoroutineContext
        get() = Dispatchers.Main + job

    override fun getName(): String = "GoogleSignInModule"

    @ReactMethod
    fun signIn(promise: Promise) {
        val activity: Activity = currentActivity ?: run {
            promise.reject("ACTIVITY_NULL", "Activity is null")
            return
        }
        Log.d("GoogleSignIn", "Activity = $activity")

        launch {
            try {
                val credentialManager = CredentialManager.create(reactContext)

                val googleIdOption = GetGoogleIdOption.Builder()
                    .setFilterByAuthorizedAccounts(false)
                    .setServerClientId("439290157125-jcv78uik1l9s87j5tn76bfsmqp40i154.apps.googleusercontent.com")
                    .build()

                val request = GetCredentialRequest.Builder()
                    .addCredentialOption(googleIdOption)
                    .build()

                Log.d("GoogleSignIn", "Request = $request")
                val response = credentialManager.getCredential(activity, request)
                Log.d("GoogleSignIn", "Response = $response")

                val credential = GoogleIdTokenCredential.createFrom(response.credential.data)
                val idToken = credential.idToken

                promise.resolve(idToken)
            } catch (e: Exception) {
                promise.reject("GOOGLE_SIGNIN_ERROR", e.message, e)
            }
        }
    }
}

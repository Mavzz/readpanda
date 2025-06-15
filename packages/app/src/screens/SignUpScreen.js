import { useState } from "react";
import {
  View,
  Text,
  TextInput,
  Pressable,
  Alert,
  ActivityIndicator,
} from "react-native";
import Background from "../components/Background";
import {
  primaryButton as PrimaryButton,
  ssoButton as SSOButton,
} from "../components/Button";
import { loginStyles } from "../styles/global";
import { usePost } from "../services/usePost";
import { useGet } from "../services/useGet";
import { encryptedPassword, getBackendUrl, SignUpType, googleSignUpLogin } from "../utils/Helper";
import { storage } from "../utils/storage";

const SignUp = ({ navigation }) => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [email, setEmail] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  let status, response;

  const handleSignUp = async (signUpType = "") => {
    try {
      setLoading(true);
      if (signUpType === SignUpType.Google) {
        await googleSignUp();
      } else if (signUpType === SignUpType.Email) {
        ({ status, response } = await googleSignUpLogin());
      }

      // Store the token in MMKV storage
      const userStorage = storage("user_storage");
      userStorage.set("token", response.token);
      userStorage.set("username", response.username);

      console.log("Signup Response:", response);

      if (status === 201) {
        /*const preferences = await getUserPreferences(
        response.token,
        username);*/

        ({ status, response } = await useGet(
          await getBackendUrl(
            `/user/preferences?username=${response.username}`
          ),
          {
            Authorization: `Bearer ${response.token}`,
          }
        ));

        console.log("Preferences Response:", response);

        navigation.popTo("InterestScreen", {
          username: username,
          preferences: response,
        });
      } else if (status === 403) {
        setLoading(false);
        Alert.alert(
          "Sign Up Failed",
          "Forbidden: You do not have permission to access this resource."
        );
      } else if (status === 500) {
        setLoading(false);
        Alert.alert(
          "Sign Up Failed",
          "Internal Server Error: Please try again later."
        );
      } else if (status === 401) {
        setLoading(false);
        Alert.alert(
          "Sign Up Failed",
          "Unauthorized: Please check your authentication token."
        );
      } else if (status === 400) {
        setLoading(false);
        Alert.alert(
          "Sign Up Failed",
          "Bad Request: Please check the data sent to the server."
        );
      } else if (status === 409) {
        setLoading(false);
        Alert.alert("Sign Up Failed", "User already exists. Please try again.");
      } else {
        setLoading(false);
        Alert.alert("Sign Up Failed", "An error occurred. Please try again.");
      }
    } catch (error) {
      setLoading(false);
      Alert.alert("Login failed", "An error occurred. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const emailSignUp = async () => {
    if (!username || !password || !confirmPassword) {
      setLoading(false);
      Alert.alert("Sign Up failed", "Please fill in all fields");
      return;
    }

    if (password !== confirmPassword) {
      setLoading(false);
      Alert.alert("Sign Up failed", "Passwords do not match");
      return;
    }

    ({ status, response } = await usePost(await getBackendUrl("/signup"), {
      username,
      password: encryptedPassword(password),
      email,
    }));

    //const response = await signUpUser(username, password, email);
    //console.log("Sign Up Response:", response);
  };

  return (
    <Background>
      <View style={loginStyles.container}>
        <Text style={loginStyles.mainTitletext}>Sign Up</Text>
        <View
          style={{
            flexDirection: "row",
            alignItems: "center",
            marginBottom: 15,
          }}
        >
          <Text style={loginStyles.title}>Already have an account?</Text>
          <Pressable onPress={() => navigation.popTo("Login")}>
            <Text style={loginStyles.signUpText}> Login!</Text>
          </Pressable>
        </View>
        <TextInput
          style={loginStyles.input}
          placeholder="Username"
          value={username}
          onChangeText={setUsername}
          placeholderTextColor="#6c757d"
          autoCapitalize="none"
        />
        <TextInput
          style={loginStyles.input}
          placeholder="Email"
          value={email}
          onChangeText={setEmail}
          keyboardType="email-address"
          placeholderTextColor="#6c757d"
          autoCapitalize="none"
        />
        <TextInput
          style={loginStyles.input}
          placeholder="Password"
          value={password}
          onChangeText={setPassword}
          placeholderTextColor="#6c757d"
          autoCapitalize="none"
          secureTextEntry
        />
        <TextInput
          style={loginStyles.input}
          placeholder="Confirm Password"
          value={confirmPassword}
          onChangeText={setConfirmPassword}
          placeholderTextColor="#6c757d"
          autoCapitalize="none"
          secureTextEntry
        />
        {loading ? (
          <ActivityIndicator size="large" color="#0000ff" />
        ) : (
          <>
            <PrimaryButton
              onPress={() => handleSignUp("Email")}
              title="Sign Up"
            />
            <View style={loginStyles.dividerContainer}>
              <View style={loginStyles.divider} />
              <Text style={loginStyles.orText}>or</Text>
              <View style={loginStyles.divider} />
            </View>

            <SSOButton
              onPress={() => handleSignUp("Google")}
              title="Sign Up with Google"
            />
          </>
        )}
      </View>
    </Background>
  );
};

export default SignUp;

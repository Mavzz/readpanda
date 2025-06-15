import { View, Text, StyleSheet } from "react-native";
import { useNavigation  } from "@react-navigation/native";
import { loginStyles } from "../styles/global";
import Background from "../components/Background";

const CurrentRead = ({ route }) => {

  const { username } = route.params;

  return (
    <Background>
      <View style={loginStyles.container}>
        <Text style={styles.welcome}>
          Welcome to the Profile Screen, {username}!
        </Text>
      </View>
    </Background>
  );
};

const styles = StyleSheet.create({
  welcome: {
    fontSize: 24,
    fontWeight: "bold",
  },
});

export default CurrentRead;

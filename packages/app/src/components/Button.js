import { View, Text, Pressable } from "react-native";
import Icon from "react-native-vector-icons/FontAwesome5";

import { loginStyles } from "../styles/global";

const primaryButton = (props) => (
  <View>
    <Pressable onPress={props.onPress} style={loginStyles.loginButton}>
      <Text style={loginStyles.loginButtonText}>{props.title}</Text>
    </Pressable>
  </View>
);

const ssoButton = (props) => (
  <View>
    <Pressable style={loginStyles.ssoButton} onPress={props.onPress}>
      <Text style={loginStyles.ssoButtonText}>{props.title}</Text>
    </Pressable>
  </View>
);

const signOutButton = (props) => (
  <View>
    <Pressable onPress={props.onPress}>
      <Icon name="sign-out-alt" size={24} color="#000000" />
    </Pressable>
  </View>
);

const iconButton = (props) => {
return (
  <View>
    <Pressable onPress={ props.onPress } style={ loginStyles.iconButton }>
      <Icon name={ props.name } size={ props.size } color={ props.color } />
    </Pressable>
  </View>
);

}

export { primaryButton, signOutButton, ssoButton, iconButton };

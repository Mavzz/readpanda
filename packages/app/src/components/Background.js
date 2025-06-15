import React from "react";
import { View, StyleSheet } from "react-native";
import { LinearGradient } from 'expo-linear-gradient';

const Background = ({ children }) => (
  <View style={styles.container}>
    <LinearGradient
      colors={["#CFFF8D", "#FFFFFF"]}
      style={styles.gradient}
      start={{ x: 0, y: 0 }}
      end={{ x: 1, y: 1 }}
    />
    <View style={styles.content}>
      {children}
    </View>
  </View>
);

const styles = StyleSheet.create({
  container: {
    flex: 1,
    position: "relative",
  },
  gradient: {
    ...StyleSheet.absoluteFillObject,
    zIndex: 0,
  },
  content: {
    flex: 1,
    zIndex: 1,
  },
});

export default Background;

import React, { useState } from "react";
import { updateUserPreferences } from "../services/api";
import { View, StyleSheet, FlatList, Button } from "react-native";
import { CheckBox } from "react-native-elements";
import { useNavigation } from "@react-navigation/native";
import Background from "../components/Background";

const InterestScreen = ({ route }) => {
  const navigation = useNavigation();
  const username = route.params.username;
  const preferencess = route.params.preferences;
  const [Interests, setInterests] = useState(preferencess);
  const [isUpdated, setIsUpdated] = useState(false);

  const togglePreference = (id) => {
    setInterests((prevPreferences) =>
      prevPreferences.map((pref) =>
        pref.preference_id === id
          ? { ...pref, preference_value: !pref.preference_value }
          : pref
      )
    );
    setIsUpdated(true);
  };

  const renderItem = ({ item }) => (
    <CheckBox
      key={item.preference_id}
      title={item.preference_type}
      checked={item.preference_value}
      onPress={() => togglePreference(item.preference_id)}
    />
  );

  React.useLayoutEffect(() => {
    navigation.setOptions({
      headerRight: () => (
        <Button
          title="Save"
          onPress={() =>
            updateUserPreferences(username, Interests, isUpdated, navigation)
          }
        />
      ),
    });
  }, [navigation, username, Interests]);

  return (
    <Background>
      <View style={styles.container}>
        <FlatList
          data={Interests}
          renderItem={renderItem}
          keyExtractor={(item) => item.preference_id.toString()}
          numColumns={2}
          columnWrapperStyle={styles.row}
        />
      </View>
    </Background>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    flexDirection: "column",
    justifyContent: "flex-start",
  },
  buttonContainer: {
    marginTop: 10,
    width: "100%",
    alignItems: "center",
  },
});

export default InterestScreen;
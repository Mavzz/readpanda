import { useState } from "react";
import { updateUserPreferences } from "../services/api";
import {
  View,
  StyleSheet,
  FlatList,
  SafeAreaView,
  Pressable,
  Text,
} from "react-native";

import { useNavigation } from "@react-navigation/native";
import Background from "../components/Background";
import { loginStyles } from "../styles/global";
import { primaryButton as PrimaryButton } from "../components/Button";
import { usePost } from "../services/usePost";
import { storage } from "../utils/storage";
import { getBackendUrl } from "../utils/Helper";

const InterestScreen = ({ route }) => {
  const navigation = useNavigation();
  const username = route.params.username;
  const Interests = route.params.preferences;
  const [interests, setInterests] = useState(Interests);
  const [isUpdated, setIsUpdated] = useState(false);
  let status, response;


  const toggleSelection = (category, preference_id) => {
    setIsUpdated(true);
    setInterests((prev) => ({
      ...prev,
      [category]: prev[category].map((item) =>
        item.preference_id === preference_id
          ? { ...item, preference_value: !item.preference_value }
          : item
      ),
    }));
  };

  const renderTag = ({ item }, category) => (
    <Pressable
      key={item.preference_id}
      onPress={() => toggleSelection(category, item.preference_id)}
      style={[styles.tag, item.preference_value && styles.tagSelected]}
    >
      <Text
        style={[
          styles.tagText,
          item.preference_value && styles.tagTextSelected,
        ]}
      >
        {item.preference_subgenre}
      </Text>
    </Pressable>
  );

  const renderCategory = ({ item: category }) => (
    <View style={styles.category} key={category}>
      <Text style={styles.categoryTitle}>{category}</Text>
      <View style={styles.tagContainer}>
        <FlatList
          data={interests[category]}
          renderItem={(item) => renderTag(item, category)}
          keyExtractor={(item) => item.preference_id.toString()}
          numColumns={3}
        />
      </View>
    </View>
  );

  const updateUserPreferences = async (username, interests, isUpdated, navigation) => {
    
    const userStorage = storage("user_storage");
    const userToken = userStorage.getString("token");
    username = userStorage.getString("username");
    console.log("User Token:", userToken);
    console.log("Username:", username);
    
    if (!isUpdated) {
      navigation.popTo("Root", { username });
    } else {
      try {
        
        ({ status, response } = await usePost(
          await getBackendUrl(`/user/preferences?username=${username}`),
          {
            username,
            preferences: interests,
          },
          {
            Authorization: `Bearer ${userToken}`,
          }
        ));
        console.log("Preferences updated successfully");
        navigation.popTo("Root", { username });
      } catch (error) {
        console.error("Preferences update failed:", error);
      }
    }
  };

  return (
    <Background>
      <View style={styles.container}>
        <SafeAreaView>
          <View style={{ flexDirection: "row" , alignItems: "center", paddingBottom: 10 }}>
            <Text
              style={{
                fontSize: 24,
                //marginBottom: 20,
                textAlign: "center",
                color: "#0A210F",
              }}
            >
              What's your
            </Text>
            <Text
              style={{
                fontSize: 24,
                //marginBottom: 20,
                color: "#E34A6F",
                fontWeight: "400",
                paddingLeft: 5,
              }}
            >
              flavour?
            </Text>
          </View>
          <FlatList
            data={Object.keys(interests)}
            renderItem={renderCategory}
            keyExtractor={(item) => item}
            contentContainerStyle={styles.flatListContent}
            ListFooterComponent={
              <PrimaryButton onPress = { () => updateUserPreferences(username, interests, isUpdated, navigation
                  )} title = "Done!" />
            }
          />
        </SafeAreaView>
      </View>
    </Background>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    paddingHorizontal: 20,
    paddingTop: 80,
  },
  flatListContent: {
    paddingBottom: 40,
  },
  header: {
    fontSize: 20,
    fontWeight: "600",
    marginVertical: 20,
    textAlign: "center",
  },
  category: {
    marginBottom: 30,
  },
  categoryTitle: {
    fontSize: 16,
    fontWeight: "700",
    marginBottom: 10,
  },
  tagContainer: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 10,
  },
  tag: {
    paddingVertical: 8,
    paddingHorizontal: 14,
    borderRadius: 20,
    backgroundColor: "#fff",
    borderColor: "#D0D0D0",
    borderWidth: 1,
    marginRight: 8,
    marginBottom: 10,
  },
  tagSelected: {
    backgroundColor: "#DDE8FF",
    borderColor: "#3366FF",
  },
  tagText: {
    color: "#333",
    fontSize: 14,
  },
  tagTextSelected: {
    color: "#3366FF",
    fontWeight: "600",
  },
  saveButton: {
    backgroundColor: "#3366FF",
    paddingVertical: 14,
    borderRadius: 30,
    alignItems: "center",
    marginTop: 10,
  },
  saveButtonText: {
    color: "#fff",
    fontSize: 16,
    fontWeight: "600",
  },
});

export default InterestScreen;

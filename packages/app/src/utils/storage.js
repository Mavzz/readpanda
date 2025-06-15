import AsyncStorage from "@react-native-async-storage/async-storage";
import { MMKV } from 'react-native-mmkv'
import { SECRET_KEY } from "@env";

export const storeToken = async (token, username) => {
  try {
    const jsonValue = JSON.stringify(token);
    await AsyncStorage.setItem(`token_${username}`, jsonValue);
  } catch (e) {
    throw new Error("Error saving data to AsyncStorage");
  }
};

export const storage = (storage_name)=> {
  const storage = new MMKV({
    id: storage_name, // Unique identifier for the storage
    encryptionKey: SECRET_KEY, // Optional: Use an encryption key for security
  });
  return storage;
}
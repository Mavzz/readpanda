//config.js
const mongoose = require("mongoose");

const connectDB = async () => {
  try {
    await mongoose.connect(
      "mongodb+srv://${process.env.MONGO_USER}:${process.env.MONGO_USER_PWD}@cluster0.vlhig1a.mongodb.net/${process.env.MONGO_DB}?retryWrites=true&w=majority"
    );
    console.log("MongoDB connected");
  } catch (err) {
    console.error(err.message);
    process.exit(1);
  }
};

module.exports = connectDB;

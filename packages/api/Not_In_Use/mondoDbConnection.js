const { MongoClient, ServerApiVersion } = require("mongodb");

exports.connectToMongoDB = async () => {

  const uri =
    "mongodb+srv://aditya:jSPDccBW0m12Oey3@cluster0.hjqo2.mongodb.net";

  // Create a MongoClient with a MongoClientOptions object to set the Stable API version
  const client = new MongoClient(uri, {
    serverApi: {
      version: ServerApiVersion.v1,
      strict: true,
      deprecationErrors: true,
    },
  });

  try {
    // Connect the client to the server	(optional starting in v4.7)
    await client.connect();
    // Send a ping to confirm a successful connection
    console.log("Connected to MongoDb");

  } catch (error) {
    console.error("Error connecting to MongoDB:", error);
    throw error;
  }

  return client;
 }

 exports.connectToaccount = async (client, accountCollectionName) => {

  const ReadPandadb = "ReadPanda";
  const db = client.db(ReadPandadb);
  const account = db.collection(accountCollectionName);

  const findOneQuery = { username: "nvaditya" };
  try {
    const findAccount = await account.findOne(findOneQuery);

    if (findAccount === null) {
      console.log("Couldn't find account.\n");
    } else {
      console.log(`Found account:\n${JSON.stringify(findOneResult)}\n`);
    }

    // add a linebreak
    console.log();
  } catch (err) {
    console.error(
      `Something went wrong trying to find the one documents: ${err}\n`
    );
  } finally {
   await client.close();
   return findAccount;
  }
 };

function validatorphoneNumber (phoneNumber) {
  let phoneNumberPattern = /^[0-9]{10}$/;
  return phoneNumberPattern.test(phoneNumber);
}

function validatorEmail(email) {
  let emailPattern =
    /[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*@(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?/;
  return emailPattern.test(email);
}

function validatorPassword(password) {
  let emailPattern = /^(?=.*\d)(?=.*[a-z])(?=.*[A-Z])(?=.*[a-zA-Z]).{8,}$/;
  return emailPattern.test(password);
}

//Schema
const account = new connectDB.Schema({
  username: {
    type: String,
    unique: true,
    required: true,
  },
  password: {
    type: String,
    required: true,
    valdate: {
      validator: validatorPassword,
      message: "Invalid Password.",
    },
  },
  phoneNumber: {
    type: Number,
    valdate: {
      validator: validatorphoneNumber,
      message: "Invalid Phone Number, Phone Number should be 10 digits.",
    },
  },
  email: {
    type: String,
    required: true,
    valdate: {
      validator: validatorEmail,
      message: "Invalid Email provided.",
    },
  },
  firstName: {
    type: String,
    required: true,
  },
  lastName: {
    type: String,
    required: true,
  },
});

//Model
const accountsRepo = connectDB.model("accounts", account);

module.exports = {
  accountsRepo,
  gridfsBucket,
};
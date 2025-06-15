const { accountsRepo } = require("../Model/Schema");
const helper = require("../Utilities/helper");
const jwt = require("jsonwebtoken");


exports.signUp = async (req, res) => {
  try {
    let userDetails = await accountsRepo.find(
      { username: req.body.username },
      { _id: 0, __v: 0 }
    );

    if ((userDetails.length) == 0) {
      const newUser = await accountsRepo.create({
        username: req.body.username,
        phoneNumber: req.body.phoneNumber,
        email: req.body.email,
        password: helper.cryptPassword(req.body.password),
        firstName: req.body.firstName,
        lastName: req.body.lastName,
      });

      res.status(201).json({
        status: "success",
        message: "User Registered with Name: " + req.body.username,
      });
    } else {
        if ((userDetails.length) > 0) {
            res.status(404).json({
                status: "fail",
                message: "User already registered",
            });
        } else {
            res.status(404).json({
                status: "fail",
                message: "There was an issue",
        });
      }
    }
  } catch (err) {
    res.status(404).json({
      status: "fail",
      message: err,
    });
  }
};

exports.loginCheck = async (req, res) => {
  try {
    let userDetails = null;
    let isPasswordMatch = helper.decryptPassword(req.body.password);
    if (isPasswordMatch) {
      userDetails = await accountsRepo.find(
        { username: req.body.username },
        { _id: 0, __v: 0 }
      );
    } else {
      res.status(400).json({
        status: "Failed",
        data: {
          message: "Password does not match",
        },
      });
    }
    const token = jwt.sign({ username: req.body.username }, "secret", {
      expiresIn: "1h",
    });

    if (userDetails.length > 0) {
      res.status(200).json({
        status: "success",
        results: userDetails.length,
        data: {
          message: "YAY!! User found!!",
          token: token,
        },
      });
    } else {
      res.status(400).json({
        status: "success",
        data: {
          message: "No userDetails available in the repo",
        },
      });
    }
  } catch (err) {
    res.status(404).json({
      status: "fail",
      message: err.message,
    });
    console.log(err);
  }
};
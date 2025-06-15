import { Platform, StyleSheet } from "react-native";

const WEB_FONT_STACK =
  'system-ui, "Segoe UI", Roboto, Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol"';

const MyTheme = {
  dark: false,
  colors: {
    primary: 'rgb(17, 214, 145)',
    background: 'rgb(242, 242, 242)',
    card: 'rgb(255, 255, 255)',
    text: 'rgb(28, 28, 30)',
    border: 'rgb(199, 199, 204)',
    notification: 'rgb(255, 69, 58)',
  },
  fonts: Platform.select({
    web: {
      regular: {
        fontFamily: WEB_FONT_STACK,
        fontWeight: '400',
      },
      medium: {
        fontFamily: WEB_FONT_STACK,
        fontWeight: '500',
      },
      bold: {
        fontFamily: WEB_FONT_STACK,
        fontWeight: '600',
      },
      heavy: {
        fontFamily: WEB_FONT_STACK,
        fontWeight: '700',
      },
    },
    ios: {
      regular: {
        fontFamily: 'System',
        fontWeight: '200',
      },
      medium: {
        fontFamily: 'System',
        fontWeight: '300',
      },
      bold: {
        fontFamily: 'System',
        fontWeight: '400',
      },
      heavy: {
        fontFamily: 'System',
        fontWeight: '500',
      },
    },
    default: {
      regular: {
        fontFamily: 'sans-serif',
        fontWeight: 'normal',
      },
      medium: {
        fontFamily: 'sans-serif-medium',
        fontWeight: 'normal',
      },
      bold: {
        fontFamily: 'sans-serif',
        fontWeight: '600',
      },
      heavy: {
        fontFamily: 'sans-serif',
        fontWeight: '700',
      },
    },
  }),
};

const loginStyles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: "center",
    padding: 50,
  },
  mainTitletext: {
    fontSize: 28,
    fontWeight: '700',
    color: '#8B0000',
    marginBottom: 10,
  },
  signUpText: {
    fontSize: 16,
    fontStyle: 'italic',
    color: '#2e7d32',
    //marginTop: 10,
  },
  title: {
    fontSize: 16,
    color: '#333',
    //marginBottom: 20,
  },
  input: {
    backgroundColor: '#fff',
    paddingVertical: 14,
    paddingHorizontal: 16,
    borderRadius: 12,
    fontSize: 16,
    marginBottom: 20,
    borderWidth: 1,
    borderColor: '#ccc',
  },
  loginButton: {
    backgroundColor: '#6a1b1a',
    paddingVertical: 14,
    borderRadius: 25,
    alignItems: 'center',
    marginBottom: 30,
  },
  loginButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
    dividerContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 20,
  },
    divider: {
    flex: 1,
    height: 1,
    backgroundColor: '#ccc',
  },
    orText: {
    marginHorizontal: 10,
    color: '#555',
  },
    ssoButton: {
    backgroundColor: '#fff',
    borderColor: '#ccc',
    borderWidth: 1,
    paddingVertical: 14,
    borderRadius: 12,
    alignItems: 'center',
  },
    ssoButtonText: {
    fontSize: 16,
    color: '#111',
  },
});

const loginStylesOld = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: "center",
    padding: 50,
  },
  mainTitletext: {
    fontSize: 24,
    //marginBottom: 20,
    color: "#E34A6F",
    fontWeight: "400",
  },
  signUpText: {
    fontSize: 16,
    color: "#548C2F",
    fontStyle: 'italic',
  },
  title: {
    fontSize: 14,
    //marginBottom: 20,
    textAlign: "center",
    color: "#0A210F",
  },
  input: {
    height: 40,
    width: "100%",
    backgroundColor: "#DBD9D8",
    borderRadius: 20,
    paddingHorizontal: 16,
    marginBottom: 12,
    color: "#E34A6F", // green text
    fontSize: 18,
    // Remove border
    borderWidth: 0,
  },
  loginButton: {
    backgroundColor: "#3B120B",
    borderRadius: 25,
    paddingVertical: 10,
    paddingHorizontal: 36,
    alignSelf: "flex-end",
    marginTop: 10,
  },
  loginButtonText: {
    color: "#f15c8e",
    fontSize: 16,
    fontWeight: "bold",
    fontStyle: "italic",
    textAlign: "center",
  },
});

export { MyTheme, loginStyles };
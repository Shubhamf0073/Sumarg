import { StyleSheet } from "react-native";

/**
 * Sign-in form styles. Colours are explicit on every surface, so the screens read the same in light and dark mode;
 * tap targets are at least 48 px and the card fits a 360 px screen.
 */
export const ui = StyleSheet.create({
  screen: { flex: 1, justifyContent: "center", padding: 16, backgroundColor: "#f1f5f9" },
  card: {
    backgroundColor: "#ffffff",
    borderRadius: 16,
    padding: 20,
    gap: 8,
    maxWidth: 480,
    width: "100%",
    alignSelf: "center",
  },
  title: { fontSize: 24, fontWeight: "700", color: "#0f172a" },
  sub: { fontSize: 14, color: "#475569", marginBottom: 8 },
  label: { fontSize: 14, fontWeight: "600", color: "#334155", marginTop: 8 },
  row: { flexDirection: "row", alignItems: "center", gap: 8 },
  grow: { flex: 1 },
  prefix: {
    fontSize: 18,
    color: "#334155",
    minHeight: 50,
    textAlignVertical: "center",
    lineHeight: 50,
  },
  input: {
    minHeight: 50,
    borderWidth: 1,
    borderColor: "#cbd5e1",
    borderRadius: 10,
    paddingHorizontal: 12,
    fontSize: 18,
    color: "#0f172a",
    backgroundColor: "#ffffff",
  },
  code: { fontSize: 28, letterSpacing: 10, textAlign: "center" },
  error: { color: "#b91c1c", fontSize: 14, marginTop: 4 },
  hint: { color: "#475569", fontSize: 13, marginTop: 12, textAlign: "center" },
  button: {
    minHeight: 50,
    borderRadius: 10,
    backgroundColor: "#1d4ed8",
    alignItems: "center",
    justifyContent: "center",
    marginTop: 12,
  },
  buttonOff: { backgroundColor: "#93c5fd" },
  buttonText: { color: "#ffffff", fontSize: 17, fontWeight: "600" },
  link: { minHeight: 48, alignItems: "center", justifyContent: "center" },
  linkText: { color: "#1d4ed8", fontSize: 15, fontWeight: "600" },
  linkOff: { color: "#94a3b8" },
});

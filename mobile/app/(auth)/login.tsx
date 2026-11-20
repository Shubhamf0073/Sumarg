import { StyleSheet, Text, View } from "react-native";
import { t } from "../../src/i18n";

export default function Login() {
  return (
    <View style={styles.box}>
      <Text style={styles.title}>{t("auth.title")}</Text>
      <Text style={styles.sub}>{t("auth.subtitle")}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  box: { flex: 1, alignItems: "center", justifyContent: "center", gap: 8, padding: 24 },
  title: { fontSize: 24, fontWeight: "700" },
  sub: { fontSize: 14, color: "#64748b", textAlign: "center" },
});

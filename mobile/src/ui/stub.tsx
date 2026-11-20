import { StyleSheet, Text, View } from "react-native";
import { type StringKey, t } from "../i18n";

/** Placeholder body for tab screens until the matching mobile-screen task lands. */
export function Stub({ title }: { title: StringKey }) {
  return (
    <View style={styles.box}>
      <Text style={styles.title}>{t(title)}</Text>
      <Text style={styles.sub}>{t("common.comingSoon")}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  box: { flex: 1, alignItems: "center", justifyContent: "center", gap: 8 },
  title: { fontSize: 20, fontWeight: "700" },
  sub: { fontSize: 14, color: "#64748b" },
});

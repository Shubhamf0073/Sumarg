import { Tabs } from "expo-router";
import { t } from "../../src/i18n";

/** Parent tab set (mirrors the demo's parentA bar). */
export default function ParentLayout() {
  return (
    <Tabs screenOptions={{ headerShown: true, tabBarActiveTintColor: "#1463ff" }}>
      <Tabs.Screen name="parent-home" options={{ title: t("tab.home") }} />
      <Tabs.Screen name="children" options={{ title: t("tab.children") }} />
      <Tabs.Screen name="tracking" options={{ title: t("tab.tracking") }} />
      <Tabs.Screen name="notifications" options={{ title: t("tab.notifications") }} />
      <Tabs.Screen name="parent-more" options={{ title: t("tab.more") }} />
    </Tabs>
  );
}

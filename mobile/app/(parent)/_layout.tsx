import { Tabs } from "expo-router";
import { useEffect } from "react";
import { registerPushToken } from "../../src/api/auth";
import { t } from "../../src/i18n";

/** Parent tab set (mirrors the demo's parentA bar). */
export default function ParentLayout() {
  // After sign-in and on every start with a stored session: FCM rotates tokens, the server keeps the newest.
  useEffect(() => {
    void registerPushToken();
  }, []);

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

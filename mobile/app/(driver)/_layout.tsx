import { Tabs } from "expo-router";
import { t } from "../../src/i18n";

/** Driver/attendant tab set (mirrors the demo's driver bar). */
export default function DriverLayout() {
  return (
    <Tabs screenOptions={{ headerShown: true, tabBarActiveTintColor: "#1463ff" }}>
      <Tabs.Screen name="driver-home" options={{ title: t("tab.home") }} />
      <Tabs.Screen name="trip" options={{ title: t("tab.trip") }} />
      <Tabs.Screen name="incidents" options={{ title: t("tab.incidents") }} />
      <Tabs.Screen name="messages" options={{ title: t("tab.messages") }} />
      <Tabs.Screen name="driver-more" options={{ title: t("tab.more") }} />
    </Tabs>
  );
}

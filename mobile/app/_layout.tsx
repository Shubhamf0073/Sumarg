import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Redirect, Slot, useSegments } from "expo-router";
import { useEffect } from "react";
import { initLang } from "../src/i18n";
import { homeGroup, LANDING } from "../src/session/principal";
import { loadSession, useSession } from "../src/session/store";

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 15_000, retry: 1, refetchOnWindowFocus: false } },
});

initLang();

/** The session's principal decides which group a signed-in user may be in. */
function Gate() {
  const { ready, session } = useSession();
  const group = useSegments()[0];

  if (!ready) return null;

  const wanted = homeGroup(session);
  if (group !== wanted) {
    return <Redirect href={LANDING[wanted]} />;
  }
  return <Slot />;
}

export default function RootLayout() {
  useEffect(() => {
    void loadSession();
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <Gate />
    </QueryClientProvider>
  );
}

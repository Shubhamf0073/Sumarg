import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useState } from "react";
import {
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  Text,
  TextInput,
  View,
} from "react-native";
import { CODE_RE, SignInError, useRequestCode, useVerifyCode } from "../../src/api/auth";
import { t } from "../../src/i18n";
import { ui } from "../../src/ui/form";

/** Seconds before "send a new code" is offered (the server allows 3 codes per mobile per hour). */
const RESEND_AFTER = 60;

/** Parent sign-in, step 2: the 6-digit SMS code. On success the root layout moves to the parent tabs. */
export default function Verify() {
  const router = useRouter();
  const { school_code = "", mobile = "" } = useLocalSearchParams<{
    school_code: string;
    mobile: string;
  }>();
  const [code, setCode] = useState("");
  const [wait, setWait] = useState(RESEND_AFTER);
  const verify = useVerifyCode();
  const resend = useRequestCode();

  useEffect(() => {
    if (wait <= 0) return;
    const id = setTimeout(() => setWait((w) => w - 1), 1000);
    return () => clearTimeout(id);
  }, [wait]);

  const valid = CODE_RE.test(code);
  const submit = (c = code) => {
    if (!CODE_RE.test(c) || verify.isPending) return;
    verify.mutate({ school_code, mobile, code: c });
  };
  const again = () =>
    resend.mutate(
      { school_code, mobile },
      {
        onSuccess: () => {
          setCode("");
          verify.reset();
          setWait(RESEND_AFTER);
        },
      },
    );
  const failure = verify.error ?? resend.error;
  const err = failure instanceof SignInError ? failure.key : null;

  return (
    <KeyboardAvoidingView
      style={ui.screen}
      behavior={Platform.OS === "ios" ? "padding" : undefined}
    >
      <View style={ui.card}>
        <Text style={ui.title}>{t("auth.enterCode")}</Text>
        <Text style={ui.sub}>
          {t("auth.codeSentTo")} +91 {mobile}
        </Text>

        <TextInput
          style={[ui.input, ui.code]}
          value={code}
          onChangeText={(v) => {
            const c = v.replace(/\D/g, "").slice(0, 6);
            setCode(c);
            if (c.length === 6) submit(c);
          }}
          keyboardType="number-pad"
          textContentType="oneTimeCode"
          autoComplete="sms-otp"
          autoFocus
          maxLength={6}
          accessibilityLabel={t("auth.enterCode")}
        />

        {err && <Text style={ui.error}>{t(err)}</Text>}

        <Pressable
          style={[ui.button, (!valid || verify.isPending) && ui.buttonOff]}
          onPress={() => submit()}
          disabled={!valid || verify.isPending}
          accessibilityRole="button"
        >
          {verify.isPending ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={ui.buttonText}>{t("auth.signIn")}</Text>
          )}
        </Pressable>

        <Text style={ui.hint}>{t("auth.noCodeHint")}</Text>
        <Pressable
          style={ui.link}
          onPress={again}
          disabled={wait > 0 || resend.isPending}
          accessibilityRole="button"
        >
          <Text style={[ui.linkText, wait > 0 && ui.linkOff]}>
            {wait > 0 ? `${t("auth.resendIn")} ${wait}s` : t("auth.resend")}
          </Text>
        </Pressable>
        <Pressable style={ui.link} onPress={() => router.back()} accessibilityRole="button">
          <Text style={ui.linkText}>{t("auth.changeNumber")}</Text>
        </Pressable>
      </View>
    </KeyboardAvoidingView>
  );
}

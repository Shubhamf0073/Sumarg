import { useRouter } from "expo-router";
import { useState } from "react";
import {
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  Text,
  TextInput,
  View,
} from "react-native";
import {
  MOBILE_RE,
  normalise,
  SCHOOL_CODE_RE,
  SignInError,
  useRequestCode,
} from "../../src/api/auth";
import { t } from "../../src/i18n";
import { ui } from "../../src/ui/form";

/** Parent sign-in, step 1: school code (from the invite) and mobile → an SMS code. */
export default function Login() {
  const router = useRouter();
  const [school, setSchool] = useState("");
  const [mobile, setMobile] = useState("");
  const request = useRequestCode();
  const input = normalise({ school_code: school, mobile });
  const valid = SCHOOL_CODE_RE.test(input.school_code) && MOBILE_RE.test(input.mobile);

  const submit = () => {
    if (!valid || request.isPending) return;
    request.mutate(input, {
      onSuccess: () => router.push({ pathname: "/(auth)/verify", params: input }),
    });
  };
  const err = request.error instanceof SignInError ? request.error.key : null;

  return (
    <KeyboardAvoidingView
      style={ui.screen}
      behavior={Platform.OS === "ios" ? "padding" : undefined}
    >
      <View style={ui.card}>
        <Text style={ui.title}>{t("auth.title")}</Text>
        <Text style={ui.sub}>{t("auth.parentSubtitle")}</Text>

        <Text style={ui.label}>{t("auth.schoolCode")}</Text>
        <TextInput
          style={ui.input}
          value={school}
          onChangeText={setSchool}
          autoCapitalize="none"
          autoCorrect={false}
          placeholder={t("auth.schoolCodeHint")}
          placeholderTextColor="#94a3b8"
          maxLength={30}
          accessibilityLabel={t("auth.schoolCode")}
        />

        <Text style={ui.label}>{t("auth.mobile")}</Text>
        <View style={ui.row}>
          <Text style={ui.prefix}>+91</Text>
          <TextInput
            style={[ui.input, ui.grow]}
            value={mobile}
            onChangeText={setMobile}
            keyboardType="phone-pad"
            textContentType="telephoneNumber"
            autoComplete="tel"
            placeholder="98765 43210"
            placeholderTextColor="#94a3b8"
            maxLength={14}
            accessibilityLabel={t("auth.mobile")}
            onSubmitEditing={submit}
          />
        </View>

        {err && <Text style={ui.error}>{t(err)}</Text>}

        <Pressable
          style={[ui.button, (!valid || request.isPending) && ui.buttonOff]}
          onPress={submit}
          disabled={!valid || request.isPending}
          accessibilityRole="button"
        >
          {request.isPending ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={ui.buttonText}>{t("auth.sendCode")}</Text>
          )}
        </Pressable>
      </View>
    </KeyboardAvoidingView>
  );
}

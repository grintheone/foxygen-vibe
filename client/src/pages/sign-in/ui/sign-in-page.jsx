import { useState } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../../features/auth";
import { routePaths } from "../../../shared/config/routes";
import { demoAccounts } from "../../../shared/model/demo-accounts";
import { PageShell } from "../../../shared/ui/page-shell";
import { AuthForm } from "../../../widgets/auth-form";
import { DemoAccountsPanel } from "../../../widgets/demo-accounts";

const initialForm = {
  username: "",
  password: "",
};

export function SignInPage() {
  const navigate = useNavigate();
  const { feedback, isSubmitting, login, setFeedback } = useAuth();
  const [form, setForm] = useState(initialForm);

  function handleChange(event) {
    const { name, value } = event.target;

    setForm((current) => ({
      ...current,
      [name]: value,
    }));
  }

  async function handleSubmit(event) {
    event.preventDefault();

    const username = form.username.trim();
    const password = form.password.trim();

    if (!username || !password) {
      setFeedback({
        tone: "error",
        message: "Username and password are required.",
      });
      return;
    }

    try {
      await login({
        username,
        password,
      });
      navigate(routePaths.dashboard);
    } catch {
      return;
    }
  }

  function autofillDemoAccount(account) {
    setForm({
      username: account.username,
      password: account.password,
    });
    setFeedback({
      tone: "success",
      message: `Loaded ${account.username}.`,
    });
  }

  return (
    <PageShell>
      <section className="grid w-full max-w-5xl gap-6 lg:grid-cols-[1.05fr_0.95fr]">
        <AuthForm
          feedback={feedback}
          form={form}
          isSubmitting={isSubmitting}
          onChange={handleChange}
          onSubmit={handleSubmit}
        />
        <DemoAccountsPanel
          accounts={demoAccounts}
          onSelect={autofillDemoAccount}
        />
      </section>
    </PageShell>
  );
}

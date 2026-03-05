import { useState } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../../features/auth";
import { routePaths } from "../../../shared/config/routes";
import { PageShell } from "../../../shared/ui/page-shell";
import { AuthForm } from "../../../widgets/auth-form";

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
        message: "Требуются имя пользователя и пароль.",
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

  return (
    <PageShell>
      <section className="w-full max-w-2xl">
        <AuthForm
          feedback={feedback}
          form={form}
          isSubmitting={isSubmitting}
          onChange={handleChange}
          onSubmit={handleSubmit}
        />
      </section>
    </PageShell>
  );
}

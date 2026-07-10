import { Link } from "react-router-dom";
import { Button } from "@/components/ui/Button";

export default function NotFound() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-3 text-center">
      <p className="text-5xl font-bold text-brand-600">۴۰۴</p>
      <p className="text-slate-500 dark:text-slate-400">صفحه‌ی مورد نظر پیدا نشد.</p>
      <Link to="/">
        <Button variant="secondary">بازگشت به خانه</Button>
      </Link>
    </div>
  );
}

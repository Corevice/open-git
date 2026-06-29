const LOCALES = [
  { locale: 'ja', text: '日本語' },
  { locale: 'en', text: 'English' },
];

export function LanguageSelector() {
  return (
    <select aria-label="Language" defaultValue="ja">
      {LOCALES.map(({ locale, text }) => (
        <option key={locale} value={locale}>
          {text}
        </option>
      ))}
    </select>
  );
}

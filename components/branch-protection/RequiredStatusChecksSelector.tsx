"use client";

import { KeyboardEvent, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type RequiredStatusChecksSelectorProps = {
  value: string[];
  onChange: (v: string[]) => void;
};

export function RequiredStatusChecksSelector({
  value,
  onChange,
}: RequiredStatusChecksSelectorProps) {
  const [inputValue, setInputValue] = useState("");

  const addContext = (raw: string) => {
    const context = raw.trim();
    if (!context || value.includes(context)) {
      return;
    }
    onChange([...value, context]);
    setInputValue("");
  };

  const removeContext = (context: string) => {
    onChange(value.filter((item) => item !== context));
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" || event.key === ",") {
      event.preventDefault();
      addContext(inputValue);
    }
  };

  return (
    <div className="space-y-2">
      <Label htmlFor="status-check-contexts">Required status check contexts</Label>
      {value.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {value.map((context) => (
            <Badge key={context} variant="secondary" className="gap-1 pr-1">
              <span>{context}</span>
              <button
                type="button"
                aria-label={`Remove ${context}`}
                className="rounded-full px-1 hover:bg-slate-300"
                onClick={() => removeContext(context)}
              >
                ×
              </button>
            </Badge>
          ))}
        </div>
      )}
      <Input
        id="status-check-contexts"
        value={inputValue}
        placeholder="Type a context and press Enter or comma"
        onChange={(event) => setInputValue(event.target.value)}
        onKeyDown={handleKeyDown}
        onBlur={() => {
          if (inputValue.trim()) {
            addContext(inputValue);
          }
        }}
      />
    </div>
  );
}

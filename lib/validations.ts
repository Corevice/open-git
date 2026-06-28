import { z } from "zod";

export const issueSchema = z.object({
  title: z
    .string()
    .min(1, "Title is required")
    .max(256, "Title must be 256 chars or fewer"),
  body: z.string().max(65536).optional(),
});

export const commentSchema = z.object({
  body: z
    .string()
    .min(1)
    .max(65536)
    .refine((v) => v.trim().length > 0, { message: "Body cannot be blank" }),
});

export const repoNameSchema = z
  .string()
  .regex(/^[A-Za-z0-9._-]{1,100}$/)
  .refine((v) => !v.startsWith(".") && !v.endsWith("."), {
    message: "Cannot start or end with a dot",
  })
  .refine((v) => !["settings", "new", "admin", "api"].includes(v.toLowerCase()), {
    message: "Name is reserved",
  });

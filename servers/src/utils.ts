import { z } from "zod";

function transformInZodSchema(properties: any): z.ZodRawShape {
    const schema: z.ZodRawShape = {};
    for (const key in properties) {
        schema[key] = toZodType(properties[key].type, properties[key].properties);
    }
    return schema;
}


function toZodType(type: string, properties: any): z.ZodType<any> {
    if (type === "string") {
        return z.string();
    }
    if (type === "number") {
        return z.number();
    }
    if (type === "boolean") {
        return z.boolean();
    }
    return z.object(transformInZodSchema(properties));
}

export { transformInZodSchema, toZodType };
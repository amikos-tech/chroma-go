/*
 * FastAPI
 *
 * No description provided (generated by Swagger Codegen https://github.com/swagger-api/swagger-codegen)
 *
 * API version: 0.1.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package swagger

type UpdateCollection struct {
	NewName     string       `json:"new_name,omitempty"`
	NewMetadata *interface{} `json:"new_metadata,omitempty"`
}
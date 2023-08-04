# {{classname}}

All URIs are relative to *http://localhost:8000*

Method | HTTP request | Description
------------- | ------------- | -------------
[**Add**](DefaultApi.md#Add) | **Post** /api/v1/collections/{collection_id}/add | Add
[**Count**](DefaultApi.md#Count) | **Get** /api/v1/collections/{collection_id}/count | Count
[**CreateCollection**](DefaultApi.md#CreateCollection) | **Post** /api/v1/collections | Create Collection
[**CreateIndex**](DefaultApi.md#CreateIndex) | **Post** /api/v1/collections/{collection_name}/create_index | Create Index
[**Delete**](DefaultApi.md#Delete) | **Post** /api/v1/collections/{collection_id}/delete | Delete
[**DeleteCollection**](DefaultApi.md#DeleteCollection) | **Delete** /api/v1/collections/{collection_name} | Delete Collection
[**Get**](DefaultApi.md#Get) | **Post** /api/v1/collections/{collection_id}/get | Get
[**GetCollection**](DefaultApi.md#GetCollection) | **Get** /api/v1/collections/{collection_name} | Get Collection
[**GetNearestNeighbors**](DefaultApi.md#GetNearestNeighbors) | **Post** /api/v1/collections/{collection_id}/query | Get Nearest Neighbors
[**Heartbeat**](DefaultApi.md#Heartbeat) | **Get** /api/v1/heartbeat | Heartbeat
[**ListCollections**](DefaultApi.md#ListCollections) | **Get** /api/v1/collections | List Collections
[**RawSql**](DefaultApi.md#RawSql) | **Post** /api/v1/raw_sql | Raw Sql
[**Reset**](DefaultApi.md#Reset) | **Post** /api/v1/reset | Reset
[**Root**](DefaultApi.md#Root) | **Get** /api/v1 | Root
[**Update**](DefaultApi.md#Update) | **Post** /api/v1/collections/{collection_id}/update | Update
[**UpdateCollection**](DefaultApi.md#UpdateCollection) | **Put** /api/v1/collections/{collection_id} | Update Collection
[**Upsert**](DefaultApi.md#Upsert) | **Post** /api/v1/collections/{collection_id}/upsert | Upsert
[**Version**](DefaultApi.md#Version) | **Get** /api/v1/version | Version

# **Add**
> bool Add(ctx, body, collectionId)
Add

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**AddEmbedding**](AddEmbedding.md)|  | 
  **collectionId** | **string**|  | 

### Return type

**bool**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Count**
> int32 Count(ctx, collectionId)
Count

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **collectionId** | **string**|  | 

### Return type

**int32**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CreateCollection**
> CreateCollectionResponse CreateCollection(ctx, body)
Create Collection

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**CreateCollection**](CreateCollection.md)|  | 

### Return type

[**CreateCollectionResponse**](CreateCollectionResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CreateIndex**
> bool CreateIndex(ctx, collectionName)
Create Index

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **collectionName** | **string**|  | 

### Return type

**bool**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Delete**
> []string Delete(ctx, body, collectionId)
Delete

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**DeleteEmbedding**](DeleteEmbedding.md)|  | 
  **collectionId** | **string**|  | 

### Return type

[**[]string**](array.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteCollection**
> DeleteCollection(ctx, collectionName)
Delete Collection

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **collectionName** | **string**|  | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Get**
> CollectionData Get(ctx, body, collectionId)
Get

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**GetEmbedding**](GetEmbedding.md)|  | 
  **collectionId** | **string**|  | 

### Return type

[**CollectionData**](CollectionData.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetCollection**
> Collection GetCollection(ctx, collectionName)
Get Collection

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **collectionName** | **string**|  | 

### Return type

[**Collection**](Collection.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetNearestNeighbors**
> QueryResult GetNearestNeighbors(ctx, body, collectionId)
Get Nearest Neighbors

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**QueryEmbedding**](QueryEmbedding.md)|  | 
  **collectionId** | **string**|  | 

### Return type

[**QueryResult**](QueryResult.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Heartbeat**
> map[string]float64 Heartbeat(ctx, )
Heartbeat

### Required Parameters
This endpoint does not need any parameter.

### Return type

**map[string]float64**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListCollections**
> []Collection ListCollections(ctx, )
List Collections

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**[]Collection**](Collection.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RawSql**
> RawSql(ctx, body)
Raw Sql

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**RawSql**](RawSql.md)|  | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Reset**
> bool Reset(ctx, )
Reset

### Required Parameters
This endpoint does not need any parameter.

### Return type

**bool**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Root**
> map[string]int32 Root(ctx, )
Root

### Required Parameters
This endpoint does not need any parameter.

### Return type

**map[string]int32**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Update**
> bool Update(ctx, body, collectionId)
Update

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**UpdateEmbedding**](UpdateEmbedding.md)|  | 
  **collectionId** | **string**|  | 

### Return type

**bool**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateCollection**
> UpdateCollection(ctx, body, collectionId)
Update Collection

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**UpdateCollection**](UpdateCollection.md)|  | 
  **collectionId** | **string**|  | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Upsert**
> bool Upsert(ctx, body, collectionId)
Upsert

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**AddEmbedding**](AddEmbedding.md)|  | 
  **collectionId** | **string**|  | 

### Return type

**bool**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Version**
> string Version(ctx, )
Version

### Required Parameters
This endpoint does not need any parameter.

### Return type

**string**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


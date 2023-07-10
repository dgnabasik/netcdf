// AuthenticationForm.tsx   https://mailboxlayer.com/documentation
import React, { useState, useEffect, useRef, forwardRef } from "react";
import { useForm } from "react-hook-form";
//import {  FieldValues, useFormContext,  SubmitHandler } from 'react-hook-form';
import "./css/authenticationForm.css";
import validator from "validator";
import UserGroups from "./UserGroups";

// https://email-checker.net/check  https://www.verifyemailaddress.org/
// https://www.digitalocean.com/community/tutorials/how-to-set-up-ssh-keys-on-ubuntu-20-04
// TODO: Become a member of list of Checkboxes for available Group.ID+names.
// TODO: output rdf code; button to upload.
// TODO: INSERT into GraphDB.

export interface IAuthenticationProps {
  options?: { mbox: string; maker: string; name: string;  password: string; sha1: string | number }[];
  mbox: string;
  maker: string;
  name: string;
  password: string;
  sha1: string;
  rdf: string;
}; 

function AuthenticationForm() {
  const [validEmail, setValidEmail] = useState(false);

  let userGroups = new Map<string, string>([
    ["IotTimeSeries_Authentication", "IoT Timeseries - Authentication"],
    ["IotTimeSeries_InferencePipeline", "IoT Timeseries - Inference Pipeline"],
    ["IotTimeSeries_SensorStream", "IoT Timeseries - Sensor Stream"],
    ["IotTimeSeries_RDFgraph", "IoT Timeseries - RDF Graph"],
    ["IotTimeSeries_AnalysisEngine", "IoT Timeseries - Analysis Engine"],
    ["IotTimeSeries_ChocolateSimulation", "IoT Timeseries - Chocolate Simulation"]
  ]);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<IAuthenticationProps>(); 
  
  const [errorMessage, setErrorMessage] = useState("")
  
  const inputRef = useRef<HTMLInputElement | null>(null);
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const onSubmit = (data: IAuthenticationProps) => {
    console.log(data);//<<<
    validateEmail(data.mbox);
    if (!validEmail || !validateMaker(data.maker) || !validatePassword(data.password) || !validateSha1(data.sha1)) 
    {
      console.log("onSubmit error");
      return
    }
    // upload data to GraphDB.<<<<
  };

  // async functions always return a promise.
  const validateEmail = async (value:string) => {
    const apiKey:string = "8edb0e203e0c2d1a9e87746f64de9a18";
    const apiUrl: string = "http://apilayer.net/api/check?access_key=" + apiKey + "&email=" + value + "&smtp=1&format=1";
    const response = await fetch(apiUrl);
    const mboxJson = await response.json(); //extract JSON from the http response
    const result:boolean = !mboxJson.hasOwnProperty("success");
    setValidEmail(result);
  }

  function validateMaker(value:string):boolean {  
    const result:boolean = value.length > 3;
    console.log("validateMaker: "+value+" "+result);//<<<
    return result;
  }

  function validatePassword(value:string):boolean {  
    console.log("validatePassword: "+value);//<<<
    if (validator.isStrongPassword(value, {
        minLength: 8, minLowercase: 1,
        minUppercase: 1, minNumbers: 1, minSymbols: 1
    })) {
        setErrorMessage("Password is strong!")
        return true;
      } else {
        setErrorMessage("Password is weak...")
        return false;
      }
  }

  // Default RSA key length is 3072 bits.
  function validateSha1(value:string):boolean { 
    const pieces:string[] = value.split(" ");
    const result:boolean = (pieces.length == 3 && pieces[0] != "ssh-rsa" && pieces[1].length >= 384);
    console.log("validateSha1: "+value+" "+result);//<<<
    return result;
  }

  function formatFOAF(data: IAuthenticationProps, groupMap: Map<string, string>):string {
    var foaf:string = `<rdf:RDF
      xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
      xmlns:rdfs="http://www.w3.org/2000/01/rdf-schema#"
      xmlns:foaf="http://xmlns.com/foaf/0.1/"
      xmlns:dc="http://purl.org/dc/elements/1.1/"
      xmlns:bibo="http://purl.org/ontology/bibo/"
      xmlns:dcterms="http://purl.org/dc/terms/"
      xmlns:ical="http://www.w3.org/2002/12/cal/ical#"
      xmlns:geo="http://www.w3.org/2003/01/geo/wgs84_pos#" >
  <foaf:PersonalProfileDocument rdf:about="">
    <foaf:maker rdf:resource="#${data.maker}"/>
    <foaf:primaryTopic rdf:resource="#${data.maker}"/>
  </foaf:PersonalProfileDocument>
 <foaf:Person>
   <foaf:name>${data.name}</foaf:name>
   <foaf:Organization>École des Mines de Saint-Étienne</foaf:Organization>
   <foaf:mbox>${data.mbox}</foaf:mbox>
   <foaf:sha1>${data.sha1}</foaf:sha1>
   `;

    foaf += `<!-- knows -->`;
    for (let [key, value] of groupMap) {
      foaf += `<foaf:knows rdf:resource="#` + key + `"/>`;
    } 
    foaf += `</foaf:Person>`;
    foaf += `<!-- groups -->`;
    for (let [key, value] of groupMap) {
      foaf += `<foaf:Group rdf:ID="` + key + `">`;
      foaf += `<foaf:name>` + value + `</foaf:name>`;
      foaf += `<foaf:member rdf:resource="#${data.maker}"/>`;
      foaf += `</foaf:Group>`;
    } 
    foaf += `</rdf:RDF>
    `;
    return foaf;
  }

//  const Select = forwardRef<HTMLSelectElement, { label: string } , password & ReturnType<UseFormRegister<IFormValues>>>(({ onChange, onBlur, name, label }, ref) => (
//       <select password={password} ref={ref} onChange={onChange} onBlur={onBlur}>
const Select = forwardRef<HTMLSelectElement, IAuthenticationProps> (({ options, password }, ref) => {  
    return (
      <>
      <label>password label</label>
      <select data-password={password} ref={ref} >
      {options?.map((op) => (<option value={op.password}>{op.password}</option>))}
      </select>
      </>
    );
  });

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="hook">
      <label className="hook__text">emse.fr Email</label>
      <input
        type="email"
        className="hook__input"
        {...register("mbox", { required: true, pattern: /^\S+@\S+$/i })}
      />
      {errors.mbox && (
        <p className="hook__error">emse.fr email is required and must be valid</p>
      )}

      <label className="hook__text">Unique rdf:resource Identifier such as FirstNameLastName (no dot, no space)</label>
      <input
        type="text"
        className="hook__input"
        {...register("maker", { required: true })}
      />
      {errors.maker && (
        <p className="hook__error">Identifier is required and must be valid</p>
      )}

      <label className="hook__text">Your First & Last Names</label>
      <input
        type="text"
        className="hook__input"
        {...register("name", { required: true })}
      />
      {errors.maker && (
        <p className="hook__error">Please enter your first & last names</p>
      )}

      <label className="hook__text">Password</label>
      <input
        type="password"
        className="hook__input"
        {...register("password", { required: true })}        
      />
      {errors.password && <p className="hook__error">Password is required</p>}

      <label className="hook__text">You will be granted Administrative access if you enter your complete (3 parts) public key that you generated on your computer using ssh-keygen.</label>
      <label className="hook__text">Open that file (~/.ssh/[mynewkey].pub) This public key will be added to the authentication server.</label>
      <input
        type="text"
        className="hook__input"
        {...register("sha1", { required: false })}
      />

      <UserGroups UserGroupProps={userGroups} />

      <button className="hook__button" type="submit">
        Check form
      </button>
    </form>
  );
}
//         onChange={(e) => validatePassword(e.target.value)}
export default AuthenticationForm;

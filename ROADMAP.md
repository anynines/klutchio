# Klutch Roadmap

## Q1 2025

- Networking backend/configuration foundation layer 
    - integrate networking backend to automatically set up connections between app clusters and the 
      individual service instances that are provisioned from these clusters
    - in this first step, the networking setup will not be fully automated, but will involve 
      operators performing manual steps to create the connections
- Generalize Klutch developer-facing interfaces
    - to prepare for the integration of more automation backends, Klutch's API abstractions must 
      be reworked to be service-agnostic
    - create a generalized domain model for data services
    - integrate extension points for service-specific options like database tuning options

## Q2 2025

- Automated network configuration
    - building on the Q1 networking configuration, fully automated networking configuration 
      should be part of Klutch
- Generic automation backend integration for platform operators
    - provide an easy-to-use “plug’n’play” interface to integrate generic automation backends or 
      providers
    - this will allow platform operators to easier offer new services to application developers 
      through Klutch
- Move control plane from monolithic setup to “cluster of clusters” approach
    - this may expose a “region” or “zone” like setting to application developers, allowing to 
      choose the hosting
    - it offers the possibility to host services on different infrastructure providers, each with 
      different qualities and cost, and allowing to choose based on a cost-benefit analysis
    - another benefit is to avoid overloading the control plane cluster when large numbers of 
      service instances are provided by Klutch

## Q3+ 2025

- Network policies to limit access to service instances
    - use networking features to better isolate service instances, and restrict access to them 
- Observability and statistics interface
    - gather information about provisioned service instances in a central repository
    - monitor internal workings of Klutch
    - use standardized interface like OpenTelemetry to allow adopters to feed data into existing 
      analysis platforms
- Improved Klutch installation and updates
    - with a growing number of components, packaging of Klutch becomes more difficult, and our 
      kustomize-based current installation becomes difficult to manage 
    - the new installation method will have to deal with CRD lifecycle management
